"""
FixtureInstance model
"""
import copy
import json
import logging
import time
import uuid

import transitions.core
from transitions.extensions import LockedMachine as Machine
import voluptuous
from voluptuous import Schema, Any, Required, All, Length, REMOVE_EXTRA

from ax.exceptions import AXApiInvalidParam, AXIllegalOperationException
from .common import InstanceStatus, INSTANCE_STATUSES, ReferrersMixin
from .lockmanager import ResourceLockManager
from .util import new_status_detail, humanize_error

_instance_locker = ResourceLockManager('instance')
lock_instance = _instance_locker.lock_resource

logger = logging.getLogger(__name__)

instance_schema = Schema({
    Required('id', default=lambda: str(uuid.uuid4())) : All(str, Length(min=1)),
    Required('name') : All(str, Length(min=1)),
    Required('description', default=''): str,
    Required('class_name'): str,
    Required('class_id'): str,
    Required('enabled', default=True): bool,
    Required('disable_reason', default=''): str,
    Required('owner'): str,
    Required('creator'): str,
    Required('status', default=InstanceStatus.INIT): str,
    Required('status_detail', default=None): Any(None, dict),
    Required('concurrency', default=1): int,
    Required('referrers', default=list) : list,
    Required('operation', default=None): Any(None, dict),
    Required('attributes', default=dict) : dict,
    Required('ctime', default=lambda: int(time.time()*1e6)) : int,
    Required('mtime', default=lambda: int(time.time()*1e6)) : int,
    Required('atime', default=lambda: int(time.time()*1e6)) : int,
}, extra=REMOVE_EXTRA)

class InstanceAction(object):
    """Standard actions that are available to all fixture instances"""
    CREATE = 'create'
    DELETE = 'delete'

class Event(object):
    """Fixture instance events which transitions the instance to next appropriate state"""
    CREATE = 'create'
    ACTION = 'action'
    ACTION_FAILURE = 'action_failure'
    ACTION_SUCCESS = 'action_success'
    MARK_ACTIVE = 'mark_active'
    DELETE = 'delete'
    MARK_DELETED = 'mark_deleted'

class FixtureInstance(ReferrersMixin):
    """A FixtureInstance class

    JSON representation:
    {
        "atime": 1493967064,
        "attributes": {
            "instance_type": "m3.large",
            "ip_address": "1.2.3.4"
        },
        "class_name": "test-fixture",
        "class_id": "5037865e-246d-45e5-af63-5cc347d03cda",
        "concurrency": 1,
        "creator": "test@email.com",
        "ctime": 1493967064,
        "description": "",
        "enabled": true,
        "disable_reason": "",
        "id": "c8186427-98fb-4a66-9490-da24c6af8305",
        "mtime": 1493967064,
        "name": "fix1",
        "operation": null,
        "owner": "tester@email.com",
        "referrers": [],
        "status": "init",
        "status_detail": null
    }
    """

    FIELDS = frozenset(['id', 'name', 'description', 'class_name', 'class_id', 'enabled', 'disable_reason',
                        'owner', 'creator', 'status', 'status_detail', 'concurrency', 'referrers',
                        'operation', 'attributes', 'ctime', 'mtime', 'atime'])
    TIME_FIELDS = frozenset(['ctime', 'mtime', 'atime'])
    JSON_FIELDS = frozenset(['status_detail', 'referrers', 'operation', 'attributes'])
    NULL_FIELDS = frozenset(['status_detail', 'operation'])
    MUTABLE_FIELDS = frozenset(['name', 'description', 'enabled', 'disable_reason', 'status', 'concurrency', 'attributes'])
    CREATE_FIELDS = MUTABLE_FIELDS | frozenset(['class_id', 'class_name', 'owner', 'creator'])

    def __init__(self, name, status):
        """FixtureInstance object model"""
        self.id = None
        self.name = name
        self.description = None
        self.class_name = None
        self.class_id = None
        self.enabled = None
        self.disable_reason = None
        self.owner = None
        self.creator = None
        self.status_detail = None
        self.concurrency = None
        self.referrers = None
        self.operation = None
        self.attributes = None
        self.ctime = None
        self.mtime = None
        self.atime = None
        self.machine = InstanceStateMachine(name, status)

    def __str__(self):
        return 'Instance {} (id:{} status:{})'.format(self.name, self.id, self.status)

    @property
    def status(self):
        """Returns the instance status based on state machine"""
        return self.machine.state

    def is_reservable(self, service_id=None, log=True):
        """Returns whether or not the instance is able to be reserved
        :param service_id: if service_id is supplied, checks list of referrers to see if it is already in the list before examining concurrency
        :param log: if True, will log the reason the instance cannot be reserved
        """
        if not self.enabled:
            if log:
                logger.warning("%s unable to be reserved: instance disabled", self)
            return False
        if self.status != InstanceStatus.ACTIVE:
            if log:
                logger.warning("%s unable to be reserved: status: %s. status_detail: %s", self, self.status, self.status_detail)
            return False
        if self.concurrency > 0:
            if service_id is not None and self.has_referrer(service_id):
                return True
            if len(self.referrers) >= self.concurrency:
                if log:
                    logger.warning("%s unable to be reserved: %s/%s slots in use", self, len(self.referrers), self.concurrency)
                return False
        return True

    def transition_state(self, event, status_detail_message):
        """Attemps to transition the fixture instance to next appropriate state based the event"""
        try:
            getattr(self.machine, event)()
        except transitions.core.MachineError as err:
            raise AXIllegalOperationException(err.value)
        self.status_detail = new_status_detail(self.status, status_detail_message)
        newtime = int(time.time() * 1e6)
        self.atime = newtime
        self.mtime = newtime
        if event in [Event.ACTION_FAILURE, Event.ACTION_SUCCESS]:
            # clear operation after job finished
            self.operation = None

    def json(self, preserve_time=False, fields=None):
        """JSON serializable dictionary of a fixture instance
        :param preserve_time: if True, leaves time fields in nanoseconds (preferred by AXDB), else seconds (preferred by API)"""
        res = {}
        for field in FixtureInstance.FIELDS:
            if fields and field not in fields:
                continue
            res[field] = getattr(self, field)
        if not preserve_time:
            for field in FixtureInstance.TIME_FIELDS:
                if field in res:
                    res[field] = int(res[field] / 1e6)
        return res

    def axdbdoc(self):
        """Normalizes json document for storage to axdb"""
        doc = self.json(preserve_time=True)
        FixtureInstance.serialize_axdbdoc(doc)
        return doc

    def requestdoc(self):
        """Returns the flattened dictionary used during fixture assignments"""
        doc = copy.deepcopy(self.attributes)
        doc['id'] = self.id
        doc['name'] = self.name
        doc['class'] = self.class_name
        return doc

    def augment_updates(self, updates):
        """Accepts a dictionary of updates and augments the updates to the current instance, also performing validation of instance level fields"""
        updates = copy.deepcopy(updates)
        if 'status' in updates:
            new_status = updates.pop('status')
            if self.status != new_status:
                if new_status == InstanceStatus.ACTIVE:
                    self.transition_state(Event.MARK_ACTIVE, "")
                elif new_status == InstanceStatus.DELETED:
                    self.transition_state(Event.MARK_DELETED, "")
                else:
                    raise AXIllegalOperationException("Cannot change status to {}".format(new_status))

        new_attributes = updates.pop('attributes', {})

        # Validates the top level fields, but not attributes (we popped the 'attributes' from the updates)
        for field, val in updates.items():
            setattr(self, field, val)
        try:
            instance_schema(self.json())
        except voluptuous.Error as err:
            raise AXApiInvalidParam(humanize_error(str(err)))

        # Set attributes so that instance attributes can be validated later against the class schema
        for attr_name, val in new_attributes.items():
            self.attributes[attr_name] = val

    @classmethod
    def discard_extra_fields(cls, payload, acceptable_fields=None):
        """Removes any fields that are deemed not modifiable"""
        if acceptable_fields is None:
            acceptable_fields = cls.MUTABLE_FIELDS
        stripped_payload = {}
        for key, val in payload.items():
            if key in acceptable_fields:
                stripped_payload[key] = val
        return stripped_payload

    @classmethod
    def from_create(cls, payload):
        """Returns a FixtureInstance from a create payload, generating a new UUID
        :param payload: fixture payload dictionary
        """
        payload = cls.discard_extra_fields(payload, acceptable_fields=FixtureInstance.CREATE_FIELDS)
        try:
            payload = instance_schema(payload)
        except voluptuous.Error as err:
            raise AXApiInvalidParam(humanize_error(str(err)))
        instance = FixtureInstance(payload['name'], InstanceStatus.INIT)
        for field in FixtureInstance.FIELDS:
            if field == 'status':
                continue
            setattr(instance, field, payload[field])
        # normalize attributes
        instance.attributes = {}
        for attr_name, val in payload['attributes'].items():
            instance.attributes[attr_name.lower()] = val
        return instance

    @classmethod
    def deserialize_axdbdoc(cls, doc):
        """Deserializes a axdb dictionary to a FixtureInstance object"""
        instance = FixtureInstance(doc['name'], doc['status'])
        for field in cls.FIELDS:
            if field == 'status':
                continue
            assert hasattr(instance, field)
            if field in cls.NULL_FIELDS:
                doc[field] = doc[field] or None
            if field in cls.JSON_FIELDS:
                if doc[field] is not None:
                    setattr(instance, field, json.loads(doc[field]))
            else:
                setattr(instance, field, doc[field])
        return instance

    @classmethod
    def deserialize_mongodoc(cls, doc):
        """Deserializes a mongodb dictionary to a FixtureInstance object"""
        instance = FixtureInstance(doc['name'], doc['status'])
        for field in cls.FIELDS:
            if field in ['status', 'deleted']:
                continue
            elif field == 'id':
                instance.id = doc['_id']
            else:
                setattr(instance, field, doc[field])
        return instance

    @classmethod
    def serialize_axdbdoc(cls, doc):
        """Serializes a dictionary to a json structure suitable for storing to axdb"""
        for json_field in cls.JSON_FIELDS:
            if json_field in doc:
                val = doc.get(json_field)
                if val is not None:
                    doc[json_field] = json.dumps(val)
        return doc

    def mongodoc(self):
        """Normalizes json document for storage to mongodb"""
        doc = self.json(preserve_time=True)
        doc['_id'] = doc['id']
        del doc['id']
        doc['name_lower'] = doc['name'].lower()
        # 'deleted' is a field that only exists in mongodb to support partial index against the name field
        doc['deleted'] = bool(self.status == InstanceStatus.DELETED)
        # 'available' supports efficient queries for fixture processing
        doc['available'] = self.is_reservable(log=False)
        return doc

class InstanceStateMachine(object):
    """Fixture Instance State Machine"""

    TRANSITIONS = [
        {'trigger': Event.CREATE, 'source': [InstanceStatus.INIT, InstanceStatus.CREATE_ERROR], 'dest': InstanceStatus.CREATING},
        {'trigger': Event.ACTION_FAILURE, 'source': [InstanceStatus.INIT, InstanceStatus.CREATING], 'dest': InstanceStatus.CREATE_ERROR},
        {'trigger': Event.ACTION_SUCCESS, 'source': InstanceStatus.CREATING, 'dest': InstanceStatus.ACTIVE},
        {'trigger': Event.ACTION, 'source': InstanceStatus.ACTIVE, 'dest': InstanceStatus.OPERATING},
        {'trigger': Event.ACTION_FAILURE, 'source': InstanceStatus.OPERATING, 'dest': InstanceStatus.ACTIVE},
        {'trigger': Event.ACTION_SUCCESS, 'source': InstanceStatus.OPERATING, 'dest': InstanceStatus.ACTIVE},
        {'trigger': Event.MARK_ACTIVE, 'source': [InstanceStatus.INIT, InstanceStatus.CREATE_ERROR], 'dest': InstanceStatus.ACTIVE},
        {'trigger': Event.DELETE, 'source': [InstanceStatus.CREATE_ERROR, InstanceStatus.ACTIVE, InstanceStatus.DELETE_ERROR], 'dest': InstanceStatus.DELETING},
        {'trigger': Event.ACTION_FAILURE, 'source': InstanceStatus.DELETING, 'dest': InstanceStatus.DELETE_ERROR},
        {'trigger': Event.ACTION_SUCCESS, 'source': InstanceStatus.DELETING, 'dest': InstanceStatus.DELETED},
        {'trigger': Event.MARK_DELETED, 'source': [InstanceStatus.INIT, InstanceStatus.CREATE_ERROR, InstanceStatus.ACTIVE, InstanceStatus.DELETE_ERROR], 'dest': InstanceStatus.DELETED},
    ]

    def __init__(self, name, initial_state):
        self.machine = Machine(
            name=name,
            model=self,
            states=INSTANCE_STATUSES,
            transitions=InstanceStateMachine.TRANSITIONS,
            initial=initial_state
        )
