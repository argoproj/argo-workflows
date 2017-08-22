"""
FixtureManager service
"""
import collections
import io
import json
import logging
import tarfile
import threading
import time

import pymongo

from ax.devops.axdb.axdb_client import AxdbClient
from ax.exceptions import AXIllegalArgumentException, AXIllegalOperationException, AXApiResourceNotFound, \
    AXApiInvalidParam, AXTimeoutException
from ax.devops.axdb.axops_client import AxopsClient
from ax.devops.kafka.kafka_client import EventNotificationClient
from ax.notification_center import FACILITY_FIXTUREMANAGER, CODE_SYSTEM_FIXTURE_TEMPLATE_DISCONNECTED, CODE_SYSTEM_FIXTURE_INVALID_ATTRIBUTES

from . import common
from .common import ServiceStatus, InstanceStatus, ATTRIBUTE_ARTIFACT_NAME, FixtureClassStatus
from .instance import FixtureInstance, InstanceAction, Event, lock_instance, _instance_locker
from .fixtureclass import FixtureClass, compare_fixture_classes
from .requestproc import FixtureRequestProcessor
from .util import substitute_attributes, TimerThread, new_status_detail, pretty_json
from .volume import FixVolumeManager

logger = logging.getLogger(__name__)

QUERY_OPERATORS = frozenset(['lt', 'lte', 'gt', 'gte', 'ne', 'eq', 'in', 'nin'])

class FixtureManager(object):
    """FixtureManager service"""

    def __init__(self, mongodb_host=None, redis_host=None, redis_db=None, axops_host=None, volume_workers=None):
        self.axdb_client = AxdbClient()
        self.mongo_client = pymongo.MongoClient(host=mongodb_host)
        self.instances = self.mongo_client[common.DB_NAME][common.INSTANCES_COLLECTION_NAME]
        # Lock to hold when inserting, updating, deleting classes to prevent duplicate class names
        self._class_lock = threading.Lock()
        # Lock to hold when inserting, updating, deleting instances to prevent duplicate instance names
        self._instance_lock = threading.Lock()

        self.axops_client = AxopsClient(host=axops_host)

        self.volumemgr = FixVolumeManager(self, volume_workers=volume_workers)
        self.reqproc = FixtureRequestProcessor(self, redis_host=redis_host, redis_db=redis_db)
        self.notification_center = EventNotificationClient(FACILITY_FIXTUREMANAGER)

        self._template_updater = None
        self._consistency_checker = None

    def create_fixture_instance(self, instance_dict):
        """Creates a new fixture and adds it to the fixture pool

        :param instance_dict: dictionary representation of the fixture instance entity
        """
        class_name = instance_dict.get('class_name')
        if not class_name:
            raise AXApiInvalidParam("'class_name' must be supplied during fixture creation")
        fix_class = self.get_fixture_class(name=class_name)
        mark_active = instance_dict.get('status') == InstanceStatus.ACTIVE
        instance_dict['class_id'] = fix_class.id
        instance_dict['class_name'] = fix_class.name
        instance = FixtureInstance.from_create(instance_dict)
        # Supply any default values if omitted from payload
        for attr_name, attr_def in fix_class.attributes.items():
            if attr_name in instance.attributes or 'default' not in attr_def:
                continue
            instance.attributes[attr_name] = attr_def['default']
        instance.attributes = fix_class.validate(instance.attributes)
        if mark_active:
            # We allow instances to be created in 'active' state in order to bypass the create step
            instance.transition_state(Event.MARK_ACTIVE, "")
        with self._instance_lock:
            self._verify_unique_instance_name(instance.name)
            self.axdb_client.create_fixture_instance(instance.axdbdoc())
            self.instances.insert_one(instance.mongodoc())

        if instance.status != InstanceStatus.ACTIVE:
            try:
                instance = self.perform_fixture_instance_action(instance.id, InstanceAction.CREATE, user=instance.creator)
            except Exception as err:
                with lock_instance(instance.id):
                    instance = self.get_fixture_instance(id=instance.id)
                    if instance.status == InstanceStatus.INIT:
                        instance.transition_state(Event.ACTION_FAILURE, "Failed to submit create action: {}".format(err))
                        self._persist_instance_updates(instance)
                raise
        logger.debug("Initialized fixture: %s\n%s", instance, pretty_json(instance.json()))
        return instance

    def _verify_unique_instance_name(self, name):
        """Verifies uniqueness of instance name, or raises AXApiInvalidParam if it already exists"""
        existing = self.instances.find_one({'name': name, 'deleted': False})
        if existing:
            raise AXApiInvalidParam("Fixture instance with name '{}' already exists".format(name))

    def perform_fixture_instance_action(self, instance_id, action_name, user=None, arguments=None):
        """Launches the necessary workflow to perform fixture instance action"""
        with lock_instance(instance_id):
            instance = self.get_fixture_instance(id=instance_id)
            fix_class = self.get_fixture_class(id=instance.class_id)
            action = fix_class.actions.get(action_name, None)
            if not action and action_name not in [InstanceAction.CREATE, InstanceAction.DELETE]:
                raise AXApiInvalidParam("Action '{}' not defined for class '{}'".format(action_name, fix_class.name))

            status_detail_message = "Performing '{}' action".format(action_name)
            if user:
                status_detail_message += " by {}".format(user)
            if action_name == InstanceAction.CREATE:
                instance.transition_state(Event.CREATE, status_detail_message)
            elif action_name == InstanceAction.DELETE:
                instance.transition_state(Event.DELETE, status_detail_message)
            else:
                instance.transition_state(Event.ACTION, status_detail_message)

            if not action:
                # Short circuit the create/delete workflow since no action is defined
                logger.info("Short-circuiting %s action: action not defined", action_name)
                if action_name == InstanceAction.CREATE:
                    instance.transition_state(Event.ACTION_SUCCESS, "")
                elif action_name == InstanceAction.DELETE:
                    instance.transition_state(Event.ACTION_SUCCESS, "Deleted by {}".format(user or "user"))
                self._persist_instance_updates(instance)
                if instance.status == InstanceStatus.ACTIVE:
                    self.reqproc.trigger_processor()
                return instance

            logger.info("Launching %s workflow (template: %s) against instance %s", action_name, action['template'], instance)
            service = {
                'name': 'Fixture {} {}'.format(instance.name, action_name),
                'notifications': [{
                    "whom": ["fixturemanager"],
                    "when": ["on_completion"]
                }],
                'annotations': {
                    'instance_id': instance.id,
                    'action': action_name
                },
                'fixtures': {
                    instance.id: instance.requestdoc()
                }
            }
            if user:
                service['user'] = user
                service['notifications'].append({
                    "whom": ["submitter"],
                    "when": ["on_failure", "on_success"]
                })
            # Attempt to get the most up-to-date version of the service template to submit for the action.
            # The template may no longer exist (e.g. the repo was disconnected). If we are unable to find the template
            # fall back to the version we cache with the class.
            service_template = self.axops_client.get_template(fix_class.repo, fix_class.branch, action['template'])
            if service_template:
                service['template_id'] = service_template['id']
            else:
                logger.warning("Service template '%s' not found in repo %s branch %s. Falling back to cached version", action['template'], fix_class.repo, fix_class.branch)
                service['template'] = fix_class.action_templates[action['template']]

            default_args = action.get('arguments', {})
            if default_args or arguments:
                service['arguments'] = default_args
                if arguments is not None:
                    service['arguments'].update(arguments)
                for arg_name, arg_val in service['arguments'].items():
                    service['arguments'][arg_name] = substitute_attributes(arg_val, instance)

            logger.debug("Submitting service:\n%s", pretty_json(service))
            service = self.axops_client.create_service(service)
            instance.operation = {
                'id': service['id'],
                'name': service['name'],
            }
            logger.info("Service created: %s", instance.operation)
            self._persist_instance_updates(instance)
            return instance

    def process_action_result(self, service):
        """Callback which we or axops invokes upon completion of an action job to transition the instance state. Payload will be the entire service object
        Example payload:
        {
            "name": "Fixture db1 backup",
            "id": "c571a637-31ed-11e7-983d-0a58c0a88d02",
            "status": 0,
            "annotations": {
                "instance_id": "cb6adbc0-bc2e-4898-ba6f-433f335ef0fa",
                "action": "backup"
            },
            "user": "tester@email.com"
        }
        """
        logger.info("Notified of action result:\n%s", pretty_json(service))
        annotations = service.get('annotations', {})
        instance_id = annotations.get('instance_id')
        action_name = annotations.get('action')
        if not instance_id:
            logger.warning("Received action result without 'instance_id' annotation")
            return
        with lock_instance(instance_id):
            instance = self.get_fixture_instance(id=instance_id, verify_exists=False)
            if not instance:
                logger.warning("Received action result for instance which does not exist")
                return
            if not instance.operation:
                # This theoretically could happen if we somehow process the job status twice
                # (e.g. one by axops and one by a backup GC process)
                logger.warning("%s no longer associated with job %s", instance, service['id'])
                return
            if service['id'] != instance.operation['id']:
                logger.warning("Notified of action result (job: %s) which does not match current operation (job: %s)", service['id'], instance.operation['id'])
                return
            if not ServiceStatus.completed(service['status']):
                logger.warning("Notified of incomplete job %s (status: %s)", service['id'], service['status'])
                return

            logger.info("%s completed service %s", instance, service['id'])
            if service['status'] == ServiceStatus.SUCCESS:
                status_detail_message = ""
                if instance.status == InstanceStatus.DELETING:
                    status_detail_message = "Deleted by user {}".format(service['user'])
                instance.transition_state(Event.ACTION_SUCCESS, status_detail_message)
            elif service['status'] in [ServiceStatus.CANCELLED, ServiceStatus.SKIPPED]:
                if instance.status in [InstanceStatus.CREATING, InstanceStatus.DELETING]:
                    # a canceled/skipped job for create/delete is considered create/delete failure
                    instance.transition_state(Event.ACTION_FAILURE, "")
                else:
                    # ignore disable policy for cancelled jobs, simply move it back to 'active' state
                    instance.transition_state(Event.ACTION_SUCCESS, "")
            else:
                instance.transition_state(Event.ACTION_FAILURE, "")

            self.set_enabled_from_policy(instance, service['status'], action_name)
            self.update_attributes_from_artifacts(instance, service)
            self._persist_instance_updates(instance)

        if instance.status == InstanceStatus.ACTIVE:
            # this fixture made it to active pool so trigger processor
            self.reqproc.trigger_processor()

    def set_enabled_from_policy(self, instance, service_status, action_name):
        """Sets the instance's enabled field to be True or False depending on the policy"""
        fixture_class = self.get_fixture_class(id=instance.class_id, verify_exists=False)
        if not fixture_class:
            logger.error("Fixture class %s for instance %s no longer exists. This should not happen.", instance.class_id, instance)
            return

        action_def = fixture_class.actions.get(action_name) or {}
        if service_status == ServiceStatus.SUCCESS:
            policy = action_def.get('on_success', '')
        elif service_status == ServiceStatus.FAILED:
            policy = action_def.get('on_failure', '')
        else:
            return

        new_enabled_state = None
        if 'disable' in policy:
            new_enabled_state = False
        elif 'enable' in policy:
            new_enabled_state = True
        else:
            return

        if instance.enabled != new_enabled_state:
            # If we went from enabled->disabled set the disable_reason
            # If we went from disabled->enabled clear the disable_reason
            # If it was already disabled (disable->disable), the original disable_reason will be presevered
            instance.enabled = new_enabled_state
            action_result = 'failed' if service_status == ServiceStatus.FAILED else 'successful'
            if new_enabled_state:
                instance.disable_reason = ''
                logger.info("Re-enabled %s from %s '%s' action", instance, action_result, action_name)
            else:
                logger.info("Disabled %s from %s '%s' action", instance, action_result, action_name)
                instance.disable_reason = "Disabled from {} '{}' action".format(action_result, action_name)
        else:
            logger.info("%s 'enabled' state is already: %s. No update to disable_reason", instance, instance.enabled)

    def update_attributes_from_artifacts(self, instance, service):
        """Checks the completed service for artifact named 'attributes'
        If exists, loads the json and updates the instance with attribute values"""
        try:
            artifacts = self.axops_client.search_artifacts({'workflow_id': service['id'], 'name': ATTRIBUTE_ARTIFACT_NAME})
            if not artifacts:
                logger.info("Job did not have '%s' artifact. Skipping attribute parsing", ATTRIBUTE_ARTIFACT_NAME)
                return
            logger.info("Found '%s' artifact for job %s. Downloading", ATTRIBUTE_ARTIFACT_NAME, service['id'])
            attr_json = self.download_artifact_json(artifacts[0]['artifact_id'])
            if not attr_json:
                detail = {
                    "Job": service['name'],
                    "Job ID": service['id'],
                    "Message": "The '{}' artifact did not contain valid json".format(ATTRIBUTE_ARTIFACT_NAME)
                }
                self.notification_center.send_message_to_notification_center(CODE_SYSTEM_FIXTURE_INVALID_ATTRIBUTES, detail=detail)
                return

            # Found attribute json
            logger.info("Updating %s attributes from artifact values: %s", instance, attr_json)
            fix_class = self.get_fixture_class(name=instance.class_name)
            bad_values = {}
            num_good_values = 0
            for attr_name, val in attr_json.items():
                if attr_name not in fix_class.attributes:
                    logger.info("Ignoring attribute '%s': not defined in class", attr_name)
                    continue
                try:
                    validated_val = fix_class.validate_attribute(attr_name, val)
                    instance.attributes[attr_name] = validated_val
                    num_good_values += 1
                except AXApiInvalidParam as err:
                    logger.info("Attribute '%s' had invalid value: %s (err: %s)", attr_name, val, err)
                    bad_values[attr_name] = val
            if bad_values:
                message = "The '{}' json had invalid values for attributes: {}".format(ATTRIBUTE_ARTIFACT_NAME, ', '.join(bad_values.keys()))
                logger.info(message)
                detail = {
                    "Job": service['name'],
                    "Job ID": service['id'],
                    "Message": message,
                    "Invalid Attributes": json.dumps(bad_values)
                }
                self.notification_center.send_message_to_notification_center(CODE_SYSTEM_FIXTURE_INVALID_ATTRIBUTES, detail=detail)
                return

            logger.info("Updated %s/%s attribute values for %s",
                        num_good_values, num_good_values + len(bad_values), instance)
        except Exception:
            logger.exception("Failed to update attributes from artifact")

    def download_artifact_json(self, artifact_id):
        """Returns a dictionary from the attribute artifact json file, or None if json was invalid"""
        try:
            response = self.axops_client.download_artifact(artifact_id)
            data = io.BytesIO(response.raw.read())
            tar = tarfile.open(mode='r:*', fileobj=data)
            for member in tar.getnames():
                attr_str = tar.extractfile(member).read().decode('utf-8')
                try:
                    return json.loads(attr_str)
                except ValueError:
                    logger.warning("Artifact %s did not have valid json", member)
        except Exception:
            logger.exception("Could not download artifact")
        return None

    def get_fixture_instance(self, id=None, name=None, verify_exists=True):
        """Returns a single, unique fixture by name or id

        >>> fixmgr.get_fixture_instance(name='linux-01')
        """
        query = {}
        if id:
            query['_id'] = id
        if name:
            query['name_lower'] = name
        if not query:
            raise AXIllegalArgumentException("No query filters supplied")
        fix_doc = self.instances.find_one(query)
        if not fix_doc:
            if verify_exists:
                raise AXApiResourceNotFound("No instances found matching: {}".format(id or name))
            return None
        return FixtureInstance.deserialize_mongodoc(fix_doc)

    def update_fixture_instance(self, updates, user=None):
        """Update one or more fields of a fixture"""
        instance_id = updates.get('id')
        if not instance_id:
            raise AXApiInvalidParam("Fixture instance id not supplied")

        with lock_instance(instance_id):
            instance = self.get_fixture_instance(id=instance_id)
            updates = FixtureInstance.discard_extra_fields(updates)
            if 'enabled' in updates:
                toggled_enabled = updates['enabled'] != instance.enabled
                if toggled_enabled:
                    if updates['enabled']:
                        updates['disable_reason'] = ''
                    else:
                        if 'disable_reason' not in updates:
                            updates['disable_reason'] = "Manually disabled by {}".format(user or "user")

            instance.augment_updates(updates)
            if 'attributes' in updates:
                # only validate attributes if they were supplied
                # This allows enable/disable marking active/deleted without failing attribute validation
                fix_class = self.get_fixture_class(id=instance.class_id)
                logger.info("Validating attributes against %s schema", fix_class)
                instance.attributes = fix_class.validate(instance.attributes)

            self._persist_instance_updates(instance)
            logger.debug("Updated %s: %s", instance, updates)
            self.reqproc.trigger_processor()
            return instance

    def _persist_instance_updates(self, instance):
        """Helper to persist updates fixture instance to both the cache and axdb. May raise AXIllegalOperationException if duplicate name is detected"""
        assert instance.id in _instance_locker.resource_locks, "update attempted without acquiring resource lock"
        pre_update_doc = self.instances.find_one({'_id' : instance.id})
        now = int(time.time() * 1e6)
        instance.mtime = now
        try:
            self.instances.replace_one({'_id' : instance.id}, instance.mongodoc())
        except pymongo.errors.DuplicateKeyError:
            raise AXApiInvalidParam("Fixture instance with name '{}' already exists".format(instance.name))
        try:
            self.axdb_client.update_fixture_instance(instance.axdbdoc())
        except Exception:
            logger.exception("Failed to persist updates for %s. Undoing cache update", instance)
            self.instances.replace_one({'_id' : instance.id}, pre_update_doc)
            raise

    def _normalize_query(self, query):
        """Normalizes a fixture API query to mongodb query
        * lowercases field names
        * replaces 'id' search with '_id'
        * replace 'name' with 'name_lower'
        * if value is a list, change query to use $in
        """
        if not query:
            return {}

        def _parse_value(value):
            """Checks and parses if supplied value is a float, bool, or int"""
            # Check if it is a boolean, int, or float value
            try:
                value = json.loads(value.lower())
                return value
            except ValueError:
                return value

        normalized = {}
        for key, val in query.items():
            if key == 'id':
                key = '_id'
            elif key in ['name']:
                key = '{}_lower'.format(key)
                val = val.lower()

            operator = None
            if isinstance(val, str):
                if ':' in val:
                    oper, new_val = val.split(':', 1)
                    if oper in QUERY_OPERATORS:
                        operator = oper
                        val = new_val
                # Check if they supplied comma separated value
                if ',' in val:
                    val = [_parse_value(v.strip()) for v in val.split(',')]
                else:
                    val = _parse_value(val)

            if operator:
                if operator in ['in', 'nin'] and not isinstance(val, list):
                    val = [val]
                val = {'$exists': True, '${}'.format(operator): val}
            elif isinstance(val, list):
                val = {'$exists' : True, '$in' : val}
            else:
                val = {'$exists' : True, '$eq' : val}

            normalized[key] = val

        return normalized

    def query_fixture_instances(self, query=None):
        """Returns a generator of fixtures based on kwarg filters
        :param query: query dictionary
        """
        query = self._normalize_query(query)
        logger.debug("Fixture query: %s", query)
        class_map = {}
        for fix_class in self.get_fixture_classes():
            class_map[fix_class.id] = fix_class.name
        for fix_doc in self.instances.find(query):
            yield FixtureInstance.deserialize_mongodoc(fix_doc)

    def delete_fixture_instance(self, instance_id, user=None):
        """Delete a fixture instance"""
        try:
            self.perform_fixture_instance_action(instance_id, InstanceAction.DELETE, user=user)
        except AXApiResourceNotFound:
            pass
        except AXIllegalOperationException:
            instance = self.get_fixture_instance(id=instance_id)
            if instance.status != InstanceStatus.DELETED:
                raise
        return instance_id

    def get_fixture_classes(self):
        """Retrieves list of fixture classes"""
        class_docs = self.axdb_client.get_fixture_classes()
        return [FixtureClass.deserialize_axdbdoc(doc) for doc in class_docs]

    def get_fixture_class(self, name=None, id=None, verify_exists=True):
        """Retrieves a FixtureClass by name or id"""
        assert any([name, id]), "name or id must be specified"
        if id:
            class_doc = self.axdb_client.get_fixture_class_by_id(id)
        else:
            class_doc = self.axdb_client.get_fixture_class_by_name(name)
        if not class_doc:
            if verify_exists:
                raise AXApiResourceNotFound("Class '{}' not found".format(name or id))
            else:
                return None
        return FixtureClass.deserialize_axdbdoc(class_doc)

    def template_to_class(self, fix_template):
        """Returns FixtureClass object from a template dictionary"""
        action_templates = {}
        actions = fix_template.get('actions', {})
        for action_name, action in actions.items():
            if action['template'] not in action_templates:
                service_template = self.axops_client.get_template(fix_template['repo'], fix_template['branch'], action['template'])
                if not service_template:
                    raise AXApiResourceNotFound("Service template for action '{}' repo: {}, branch: {}, name: '{}' not found"
                                                .format(action_name, fix_template['repo'], fix_template['branch'], action['template']))
                # Pop off some fields which we don't want to store in cache, as it
                # causes us to think the service template is constantly changing
                for field in ['cost', 'jobs_fail', 'jobs_success']:
                    service_template.pop(field, None)
                action_templates[service_template['name']] = service_template
        return FixtureClass.from_template(fix_template, action_templates)

    def upsert_fixture_class(self, template_id):
        """Creates or updates a fixture class from a fixture template

        :param template_id: fixture template id
        """
        logger.info("Creating fixture class from template %s", template_id)
        fix_template = self.axops_client.get_fixture_template(template_id)
        if not fix_template:
            raise AXApiResourceNotFound("Fixture template with id {} not found".format(template_id))
        new_class = self.template_to_class(fix_template)
        with self._class_lock:
            old_class = self.get_fixture_class(name=new_class.name, verify_exists=False)
            if not old_class:
                self.axdb_client.create_fixture_class(new_class.axdbdoc())
                logger.info("%s created:\n%s", new_class, pretty_json(new_class.json()))
                return new_class

            if old_class.repo != new_class.repo or old_class.branch != new_class.branch:
                raise AXIllegalOperationException("Fixture class '{}' already enabled from different repo/branch: {}/{}"
                                                  .format(old_class.name, old_class.repo, old_class.branch))
            logger.info("%s already exists. Checking for updates", old_class)
            return self.apply_class_changes(old_class, new_class)

    def notify_template_updates(self):
        """Periodic poller and API handler which is invoked by axops when they detect changes to the template yaml"""
        logger.info("Checking for template updates")
        for old_class in self.get_fixture_classes():
            template = self.axops_client.get_fixture_template_by_repo(old_class.repo, old_class.branch, old_class.name)
            if not template:
                errmsg = "Template for fixture class '{}' could not be found in repo: {}, branch: {}".format(old_class.name, old_class.repo, old_class.branch)
                logger.warning(errmsg)
                if old_class.status != FixtureClassStatus.DISCONNECTED:
                    old_class.status = FixtureClassStatus.DISCONNECTED
                    old_class.status_detail = new_status_detail(FixtureClassStatus.DISCONNECTED, errmsg)
                    self.axdb_client.update_fixture_class(old_class.axdbdoc(fields=['status', 'status_detail']))
                    detail = {
                        'Fixture Class': old_class.name,
                        'Repo': old_class.repo,
                        'Branch': old_class.branch,
                        'Message': "{}. Check template for errors, or reassociate with a template in another branch.".format(errmsg)
                    }
                    self.notification_center.send_message_to_notification_center(
                        CODE_SYSTEM_FIXTURE_TEMPLATE_DISCONNECTED, detail=detail)
                continue
            try:
                new_class = self.template_to_class(template)
            except AXApiResourceNotFound:
                # This should never happen since the YAML checker will ensure dependent action templates exist and are valid
                logger.error("Action templates for %s not found. Skipping change detection", old_class)
                continue
            try:
                self.apply_class_changes(old_class, new_class)
            except Exception:
                logger.exception("Failed to apply changes to %s", old_class)

    def apply_class_changes(self, old_class, new_class):
        """
        Detects schema differences between an existing/enabled class and a new class, and performs schema migration
        :param old_class: existing class
        :param new_class: new class
        """
        new_class.id = old_class.id
        if old_class.json() == new_class.json():
            logger.info("No differences detected for %s", old_class)
            return old_class

        class_rename = bool(old_class.name != new_class.name)
        _, modified_attributes, deleted_attrs = compare_fixture_classes(old_class, new_class)
        if any([modified_attributes, deleted_attrs, class_rename]):
            mutation = {}
            if class_rename:
                logger.info("Fixture class renamed from %s -> %s", old_class.name, new_class.name)
                mutation['$set'] = {'mtime' : int(time.time() * 1e6)}
                mutation['$set']['class_name'] = new_class.name

            for attr_name in modified_attributes:
                new_attr_def = new_class.attributes[attr_name]
                old_attr_def = old_class.attributes[attr_name]
                old_is_array = 'array' in old_attr_def.get('flags', '')
                new_is_array = 'array' in new_attr_def.get('flags', '')
                if old_attr_def['type'] != new_attr_def['type'] or old_is_array != new_is_array:
                    logger.warning("Data type of %s changed (%s -> %s) (array: %s -> %s). Dropping old column values",
                                   attr_name, old_attr_def['type'], new_attr_def['type'], old_is_array, new_is_array)
                    deleted_attrs.append(attr_name)

            if deleted_attrs:
                mutation['$unset'] = {}
                mutation['$set'] = {'mtime' : int(time.time() * 1e6)}
                for attr_name in deleted_attrs:
                    logger.info("Deleting class attribute: %s.%s", new_class.name, attr_name)
                    mutation['$unset']['attributes.{}'.format(attr_name)] = ""

            if mutation:
                # update all active fixtures with the change (don't touch deleted instances)
                query = {'class_id' : old_class.id, 'deleted': False}
                self._batch_update(query, mutation)
            else:
                logger.info("No instance updates to apply")

        self.axdb_client.update_fixture_class(new_class.axdbdoc())
        logger.info("%s updated:\n%s", new_class, pretty_json(new_class.json()))
        self.reqproc.trigger_processor()
        return new_class

    def _batch_update(self, query, mutation):
        """Batch apply updates to a bunch of instances one by one"""
        logger.info("Performing batch update on %s. Mutation: %s", query, mutation)
        modified = 0
        for doc in self.instances.find(query):
            with lock_instance(doc['_id']):
                pre_update_doc = self.instances.find_one({'_id' : doc['_id']})
                result = self.instances.update_one({'_id': doc['_id']}, mutation)
                assert result.modified_count == 1
                modified += 1
                updated_doc = self.instances.find_one({'_id': doc['_id']})
                instance = FixtureInstance.deserialize_mongodoc(updated_doc)
                try:
                    self.axdb_client.update_fixture_instance(instance.axdbdoc())
                except Exception:
                    logger.exception("Failed to persist updates for %s. Undoing cache update", instance)
                    self.instances.replace_one({'_id' : instance.id}, pre_update_doc)
                    raise
        logger.info("%s fixture instances modified", modified)

    def update_fixture_class(self, class_id, template_id):
        """Rebases the class to a different template id, possibly renaming the class, changing repo/branch"""
        fix_template = self.axops_client.get_fixture_template(template_id)
        if not fix_template:
            raise AXApiResourceNotFound("Fixture template with id {} not found".format(template_id))
        with self._class_lock:
            old_class = self.get_fixture_class(id=class_id)
            new_class = self.template_to_class(fix_template)
            if old_class.name != new_class.name or old_class.repo != new_class.repo or old_class.branch != new_class.branch:
                logger.info("Rebasing existing class %s from repo: %s branch: %s to template %s name: %s, repo: %s, branch: %s",
                            old_class.name, old_class.repo, old_class.branch, template_id, new_class.name, new_class.repo, new_class.branch)
            if old_class.name != new_class.name:
                # If there is a class rename, check for duplicate before allowing the rename
                duplicate_class = self.get_fixture_class(name=new_class.name, verify_exists=False)
                if duplicate_class:
                    raise AXIllegalOperationException("Fixture class '{}' already enabled from a different repo/branch: {}/{}"
                                                      .format(duplicate_class.name, duplicate_class.repo, duplicate_class.branch))
            return self.apply_class_changes(old_class, new_class)

    def delete_fixture_class(self, class_id):
        """Delete a fixture class

        :param class_id: uuid of class
        """
        with self._class_lock:
            existing_fix = self.instances.find_one({'class_id': class_id, 'status': {'$ne': InstanceStatus.DELETED}})
            if existing_fix:
                raise AXIllegalOperationException("Fixtures belonging to class {} should be deleted prior to removal"
                                                  .format(existing_fix['class_name']))
            self.axdb_client.delete_fixture_class(class_id)
            logger.info("Deleted %s", class_id)
            return class_id

    def get_summary(self, group_by=None, query=None):
        """Get availability/total summary of instances"""
        if query:
            query = self._normalize_query(query)
        else:
            query = {}
        query['deleted'] = False
        group = {'available': '$available'}
        if group_by is not None:
            for g in group_by.split(','):
                if '.' in g:
                    parts = g.split('.')
                    if parts[0] not in group:
                        group[parts[0]] = {}
                    group[parts[0]][parts[1]] = '${}.{}'.format(parts[0], parts[1])
                else:
                    group[g] = '$'+g
        results = self.instances.aggregate([
            {'$match': query},
            {'$group': {'_id': group, 'count': {'$sum': 1}}},
        ])
        summary = collections.defaultdict(lambda: {'available': 0, 'total': 0})
        def get_key_name(key):
            grouping = key['_id']
            name_arr = []
            for group_name, val in grouping.items():
                if group_name == 'available':
                    continue
                if isinstance(val, dict):
                    if val:
                        for k, v in val.items():
                            name_arr.append("{}.{}:{}".format(group_name, k, v))
                    else:
                        name_arr.append(group_name+':')
                else:
                    name_arr.append(group_name+':'+val)
            if name_arr:
                key_name = ','.join(sorted(name_arr))
                return key_name
            else:
                return 'all'
        for result in results:
            key_name = get_key_name(result)
            count = result['count']
            if result['_id']['available']:
                summary[key_name]['available'] += count
            summary[key_name]['total'] += count
        return summary

    def initdb(self):
        """Reinitalize db cache and resync from persistence layers (axdb)

        Normalize the state of the world (e.g. after service restart)"""
        logger.info("Initializing database")
        self.instances.drop()
        self.instances.create_index([('class_id', pymongo.HASHED)])
        # Creates a unique index
        self.instances.create_index(
            'name',
            unique=True,
            partialFilterExpression={'deleted' : False}
        )
        start_time = time.time()
        timeout = 60 * 5
        while not self.axops_client.ping():
            if time.time() - start_time > timeout:
                raise AXTimeoutException("Timed out ({}s) waiting for axops availability".format(timeout))
            time.sleep(3)

        for fix_doc in self.axdb_client.get_fixture_instances():
            instance = FixtureInstance.deserialize_axdbdoc(fix_doc)
            self.instances.insert_one(instance.mongodoc())

        logger.info("Database initialized")

    def release_orphaned_reservations(self):
        """Handle the case when fixture is never released properly but job is no longer running, or deployment is terminated

        Scenario:
        1) fixture is assigned (requestdb & instance both updated in axdb)
        2) fixturemanager crashes
        3) job completes/cancelled/deleted, and due to bug, axworkflowexecutor/axamm never
           notifies fixturemanager to release the fixture.
        4) fixturemanager should check job/deployment status and release fixtures
           associated with jobs and deployments that are no longer running
        """
        logger.info("Checking for orphaned jobs")
        active_jobs = set()
        active_services = set()
        active_deployments = set()
        for job_info in self.axops_client.get_services(task_only=True, is_active=True, fields='id'):
            if not ServiceStatus.completed(job_info['status']):
                active_jobs.add(job_info['id'])
        for deployment_info in self.axops_client.get_deployments(fields='deployment_id,status'):
            # NOTE: axamm will make fixture requests by the stable 'deployment_id' (which is a UUIDv5 hash
            # of the deployment name), rather than the deployment's 'id' field, which can change over time.
            if deployment_info['status'] != 'Terminated':
                active_deployments.add(deployment_info['deployment_id'])

        for fix_req in self.reqproc.requestdb.items():
            if not fix_req.assignment and not fix_req.vol_assignment:
                # Unassigned request. Still valid
                continue
            if fix_req.requester == common.FIX_REQUESTER_AXWORKFLOWADC:
                if fix_req.root_workflow_id not in active_jobs:
                    logger.warning("Found reservation for orphaned job %s:\n%s", fix_req.root_workflow_id, pretty_json(fix_req.json()))
                    self.reqproc.delete_fixture_request(fix_req.service_id)
                    continue
            elif fix_req.requester == common.FIX_REQUESTER_AXAMM:
                if fix_req.service_id not in active_deployments:
                    logger.warning("Found reservation for orphaned deployment %s:\n%s", fix_req.service_id, pretty_json(fix_req.json()))
                    # DO NOT release fixtures automatically in the case of deployments
            active_services.add(fix_req.service_id)

        for channel_key in self.reqproc.redis_client_notification.keys(regex='^(notification):'):
            service_id = channel_key.split(':', 1)[1]
            if service_id not in active_services and service_id not in active_deployments:
                logger.warning("Deleting orphaned assignment channel: %s", channel_key)
                self.reqproc.redis_client_notification.delete("notification:{}".format(service_id))

    def check_referrers_consistency(self):
        """Ensures referrer field of a fixture instances and volumes is consistent with requestdb

        Scenario:
        1) axops goes down
        2) fixture becomes assigned (updated in requestdb but fails to update axops since it is down)
        3) fixturemanager crashes
        4) During startup, fixturemanager should trust requestdb state for availability state and update axdb
        """
        logger.info("Checking for inconsistent referrers field")
        assigned_fix_reqs = set()

        # Get all assigned fixture requests. Ensure that the instance has the fixture request in it's list of referrers
        # If instance does not have the referrer, add it.
        for fix_req in self.reqproc.requestdb.items(assigned=True):
            assigned_fix_reqs.add(fix_req.service_id)
            for fix_dict in fix_req.assignment.values():
                with lock_instance(fix_dict['id']):
                    instance = self.get_fixture_instance(fix_dict['id'], verify_exists=False)
                    if not instance:
                        logger.error("Reservation held on a an instance which no longer exists")
                        continue
                    if not instance.has_referrer(fix_req.service_id):
                        logger.warning("%s was missing a referrer %s. Correcting state", instance, fix_req.service_id)
                        instance.add_referrer(fix_req.referrer())
                        self._persist_instance_updates(instance)

        # Reverse correction from above.
        # Get all instances which have referrers, ensure the referrers corresponds to a valid fixture request (which we accumulated above)
        # If the referrer field corresponds to an fixture request which does not exist, remove the referrer.
        for fix_doc in self.instances.find({'referrers': {'$exists': True, '$not': {'$size': 0}}}):
            instance = FixtureInstance.deserialize_mongodoc(fix_doc)
            to_remove = []
            for referrer in instance.referrers:
                if referrer['service_id'] not in assigned_fix_reqs:
                    logger.warning("%s has a referrer %s which no longer exists. Correcting state", instance, referrer['service_id'])
                    to_remove.append(referrer['service_id'])
            if to_remove:
                with lock_instance(instance.id):
                    instance = self.get_fixture_instance(instance.id)
                    for service_id in to_remove:
                        instance.remove_referrer(service_id)
                    self._persist_instance_updates(instance)

    def start_workers(self):
        """Start workers"""
        if self._template_updater is None:
            self._template_updater = TimerThread(10 * 60, target=self.notify_template_updates, name="TemplateUpdater", ignore_errors=True)
            self._template_updater.start()
            logger.info("Template updater thread started")
        else:
            logger.warning("Template updater thread already started.")

        if self._consistency_checker is None:
            self._consistency_checker = TimerThread(60 * 60, target=self.check_consistency, name="ConsistencyChecker", ignore_errors=True)
            self._consistency_checker.start()
            logger.info("Consistency checker thread started")
        else:
            logger.warning("Consistency checker thread already started.")

    def stop_workers(self):
        """Stop volume workers"""
        if self._template_updater:
            self._template_updater.stopped.set()
            self._template_updater = None
            logger.info("Template updater thread stopped")
        else:
            logger.info("Template updater thread already stopped")

        if self._consistency_checker:
            self._consistency_checker.stopped.set()
            self._consistency_checker = None
            logger.info("Consistency checker thread stopped")
        else:
            logger.info("Consistency checker thread already stopped")

    def check_consistency(self, raise_exception=False):
        """Periodic checker to detect and correct the following conditions:
        1) orphaned fixture requests: delete fixture request entries associated with already completed workflows and terminated deployments
           Scenario: requestor fails to release properly
        2) referrers inconsistency: adds missing referrers to instances/volumes in assigned requests, or removes referrers for removed fixture requests
           Scenario: crash between committing fixture request to database, and updating referrers for each instance/volume
        3) missed job notifications: missed notifications from axops about completed actions
           Scenario: action job completed but axops fails to notify fixturemanager
        4) purge deleted fixtures (which have been deleted for n number of days)
        """
        # Order here is important
        checkers = ['release_orphaned_reservations', 'check_referrers_consistency',
                    'check_missed_job_completion_notifications', 'expire_deleted_fixtures']
        for method in checkers:
            try:
                getattr(self, method)()
            except Exception:
                if raise_exception:
                    raise
                logger.exception("Failed to perform %s check", method)

    def check_missed_job_completion_notifications(self):
        """Checks for fixtures which are currently performing an operation (create/delete/action) to see if the job is still active"""
        logger.info("Checking for missed job completion notifications")
        #ten_min_ago = int((time.time() - 600) * 1e6)
        operating = self.instances.find({
            #'mtime': {'$lt': ten_min_ago},
            'operation' : {'$exists': True, '$ne': None}
        })

        for fix_doc in operating:
            service = self.axops_client.get_service(fix_doc['operation']['id'])
            if ServiceStatus.completed(service['status']):
                # Keep this consistent with expectation in process_action_result() and axops/service/service.go
                payload = {
                    "id": service['id'],
                    "name": service['name'],
                    "status": service['status'],
                    "annotations": service.get('annotations', {}),
                    "user": service['user']
                }
                try:
                    logger.info("Detected missed job notification: %s", payload)
                    self.process_action_result(payload)
                except Exception:
                    logger.exception("Failed to process completion event")

    def expire_deleted_fixtures(self):
        """Checks for expired fixtures and deletes them from the database"""
        logger.info("Purging instances deleted %s days ago", common.DELETED_INSTANCE_GC_DAYS)
        n_days_ago_in_usec = int((time.time() * 1e6) - (common.DELETED_INSTANCE_GC_DAYS * common.NANOSECONDS_IN_A_DAY))
        expired = self.instances.find({
            'mtime': {'$lt': n_days_ago_in_usec},
            'status': InstanceStatus.DELETED
        })
        count = 0
        for fix_doc in expired:
            logger.info("Deleting expired fixture %s (id: %s)", fix_doc['name'], fix_doc['_id'])
            self.axdb_client.delete_fixture_instance(fix_doc['_id'])
            self.instances.delete_one({'_id': fix_doc['_id']})
            count += 1
        logger.info("Expired %s instances", count)
