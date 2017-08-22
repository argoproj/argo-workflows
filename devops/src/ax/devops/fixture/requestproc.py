"""
FixtureManager submodule which handles processing of fixture requests
"""

import copy
import logging
import json
import random
import threading
import time
from contextlib import ExitStack

from . import common
from .instance import lock_instance
from .request import FixtureRequest
from .requestdb import FixtureRequestDatabase
from .util import pretty_json
from ax.devops.redis.redis_client import RedisClient, DB_RESULT
from ax.exceptions import AXException, AXApiResourceNotFound, AXApiInvalidParam, AXIllegalArgumentException

logger = logging.getLogger(__name__)

class FixtureRequestProcessor(object):

    def __init__(self, fixturemgr, redis_host=None, redis_db=None):
        self.fixmgr = fixturemgr
        self.requestdb = FixtureRequestDatabase(self.fixmgr)

        port = None
        if redis_host and ':' in redis_host:
            redis_host, port = redis_host.split(':', 2)
        self.redis_client_notification = RedisClient(host=redis_host, port=port, db=DB_RESULT)

        self.process_interval = common.DEFAULT_PROCESS_INTERVAL
        self._stop = False
        # Condition variable to notify the request processor that it should
        # wake up and process the requests
        self._process_cv = threading.Condition()
        self._events = 0
        self._request_processor_thread = None
        self._processor_lock = threading.Lock()

    @property
    def axdb_client(self):
        return self.fixmgr.axdb_client

    def create_fixture_request_mock(self, request):
        """Create a fixture request mock

        :param request: fixture request
        :return: created fixture request
        """
        def _notify_reservation_available_mock(req):
            try:
                logger.info("get mock request for %s. existing assignment=%s", req.service_id, req.assignment)
                for req_name, _ in req.requirements.items():
                    req.assignment[req_name] = {'attributes' : {'name': req_name}, 'name': req_name}
                sleep_second = random.randint(0, 20)
                logger.info("sleep %s seconds for mock request %s", sleep_second, req.service_id)
                time.sleep(sleep_second)
                logger.info("notify mock request %s", req.service_id)
                self._notify_channel(req)
            except Exception:
                logger.exception("mock exception")

        fixture_request = FixtureRequest(request)
        fixture_request.assignment = {}
        t = threading.Thread(name="mock-reply-thread-{}".format(fixture_request.service_id),
                             target=_notify_reservation_available_mock,
                             kwargs={'req': fixture_request})
        t.daemon = True
        t.start()
        return fixture_request

    def create_fixture_request(self, request):
        """Create a fixture request. For volumes, if an anonymous volume is requested, this will create
        the volume as well as reserve it.

        :param request: fixture request
        :return: created fixture request
        """
        synchronous = request.pop('synchronous', False)
        fix_req = FixtureRequest(request)
        self._validate_fixture_request(fix_req)

        if synchronous:
            # If in synchronous mode, we heed to hold the processor lock while we create the fixture request
            # and immediately process it. Otherwise, the background request processor may jump in and assign
            # it from underneath us.
            self._processor_lock.acquire()

        try:
            fix_req = self.requestdb.add(fix_req)
            if fix_req.assigned:
                logger.warning("Client made duplicate request which was already assigned. Returning existing request:\n%s", pretty_json(fix_req.json()))
            elif synchronous:
                # If fixture request was sent in synchronous mode, we attempt to assign the fixture instance(s) immediately
                assigned_req = self._process_request(fix_req)
                if not assigned_req:
                    self.requestdb.remove(fix_req.service_id)
                    raise AXApiResourceNotFound("Could not formulate resources for fixture request")
                fix_req = assigned_req
            else:
                # async request. we created the entry in the request database. now just notify the processor
                self.trigger_processor()
        finally:
            if synchronous:
                self._processor_lock.release()

        if synchronous and fix_req.vol_assignment:
            # This may raise AXTimeoutException, leaving the created & assigned fixture request.
            # The volume workers will continue to bring any volumes to active state.
            # It will be the responsibilty of the caller to decide if he should reissue the request
            # and wait longer for the volumes to become active, or give up and delete the request.
            self.fixmgr.volumemgr.wait_volumes_active(fix_req)

        if self.should_notify(fix_req):
            self._notify_channel(fix_req)
        return fix_req

    def _validate_fixture_request(self, fix_req):
        """Validates a fixture request by checking the current inventory of fixtures and volumes to ensure we can satisfy the request.
        :raises AXApiInvalidParam if request was invalid, or AXApiResourceNotFound if request was valid but could not be satisfied"""
        # Validate that the request (ensure attributes are valid)
        for requirement in fix_req.requirements.values():
            if 'class' not in requirement:
                continue
            try:
                fix_class = self.fixmgr.get_fixture_class(name=requirement['class'])
            except AXApiResourceNotFound as err:
                raise AXApiInvalidParam(err.args[0])
            for attr_name in requirement.get('attributes', {}).keys():
                if attr_name not in fix_class.attributes:
                    raise AXApiInvalidParam("Fixture class {} does not have attribute {}"
                                            .format(requirement['class'], attr_name))

        if fix_req.requirements:
            # See if assignment is even possible given current inventory of fixtures.
            # If we cannot satisfy the request, we will reject the request, since it will never be assigned
            # (unless fixtures are added)
            self._find_candidates(fix_req.requirements, validate_request=True)

        # Do the same for volumes
        if fix_req.vol_requirements:
            self.fixmgr.volumemgr.find_named_volume_candidates(fix_req, validate_request=True)
            for vol_requirement in fix_req.vol_requirements.values():
                if not vol_requirement.get('axrn'):
                    # anonymous volume request. verify storage_class exists
                    storage_class_name = vol_requirement.get('storage_class')
                    if not storage_class_name:
                        raise AXApiInvalidParam("Volume request did not supply axrn or storage class")
                    if not self.fixmgr.volumemgr.get_storage_class_by_name(storage_class_name):
                        raise AXApiInvalidParam("Storage class '{}' does not exist".format(storage_class_name))

    def reserve_instances(self, fix_req, instance_ids):
        """
        Updates the database and appends the referrer to instance's referrers

        :param fix_req: FixtureRequest object of the requestor
        :param instance_ids: list of instance ids of which to append the referrer to
        """
        logger.info("%s reserving fixture : %s", fix_req, instance_ids)
        reserved_ids = []
        with ExitStack() as stack:
            # Acquire lock on all volumes. Necessary for atomic assignment of multiple volumes
            for instance_id in instance_ids:
                stack.enter_context(lock_instance(instance_id))

            # We have a lock on all instances
            try:
                for instance_id in instance_ids:
                    instance = self.fixmgr.get_fixture_instance(id=instance_id)
                    if not instance.is_reservable(service_id=fix_req.service_id):
                        errmsg = "{} is no longer reservable".format(instance)
                        logger.error("%s\n%s", errmsg, pretty_json(instance))
                        raise AXException(errmsg)
                    modified = instance.add_referrer(fix_req.referrer())
                    if modified:
                        self.fixmgr._persist_instance_updates(instance)
                        reserved_ids.append(instance.id)
                        logger.info("Successfully reserved %s with referrer: %s", instance, fix_req)
                    else:
                        logger.warning("%s already had reservation on: %s", fix_req, instance)
            except Exception:
                logger.exception("Failed to reserve instances. Undoing reservations: %s", reserved_ids)
                try:
                    self._release_instances(fix_req, reserved_ids)
                except Exception:
                    logger.warning("Failed to release partial reservation")
                raise

    def release_instances(self, fix_req, instance_ids):
        """Releases instances. Removes the referrer from the instance

        :param fix_req: FixtureRequest object of the requestor
        :param instance_ids: list of instance ids of which to remove the referrer from
        """
        logger.info("%s releasing instances: %s", fix_req, instance_ids)
        if not instance_ids:
            return
        with ExitStack() as stack:
            # Acquire lock on all volumes. Necessary for atomic release of multiple instances
            for instance_id in instance_ids:
                stack.enter_context(lock_instance(instance_id))
            self._release_instances(fix_req, instance_ids)

    def _release_instances(self, fix_req, instance_ids):
        """Internal helper to release instances. Lock on instance is assumed"""
        for instance_id in instance_ids:
            instance = self.fixmgr.get_fixture_instance(instance_id, verify_exists=False)
            if not instance:
                logger.warning("Instance %s no longer exists", instance_id)
                continue
            instance.remove_referrer(fix_req.service_id)
            self.fixmgr._persist_instance_updates(instance)
            logger.debug("%s release of %s successful", fix_req, instance)

    def delete_fixture_request(self, service_id):
        """Releases all fixtures requested, reserved, or deployed by the specified service id.

        :param service_id: service id of the requestor
        """
        fix_req = self.requestdb.get(service_id, verify_exists=False)
        if not fix_req:
            logger.info("No fixture request found for service id %s. Ignoring deletion", service_id)
            return

        self.requestdb.remove(service_id)
        assigned_fixture_ids = [f['id'] for f in fix_req.assignment.values()]
        if assigned_fixture_ids:
            self.release_instances(fix_req, assigned_fixture_ids)
        assigned_volume_ids = [v['id'] for v in fix_req.vol_assignment.values()]
        if assigned_volume_ids:
            self.fixmgr.volumemgr.release_volumes(fix_req, assigned_volume_ids)

        self.redis_client_notification.delete(fix_req.notification_channel)

        if assigned_fixture_ids or assigned_volume_ids:
            self.trigger_processor()

        logger.info("Deleted fixture request for service id %s", service_id)
        return service_id

    def get_fixture_request(self, service_id, verify_exists=True):
        """Return fixture request for a service id"""
        return self.requestdb.get(service_id, verify_exists=verify_exists)

    def get_fixture_requests(self, assigned=None):
        """Return a list of fixture requests"""
        return self.requestdb.items(assigned=assigned)

    def _find_candidates(self, requirements, validate_request=False):
        """For each fixture requirement, queries the fixture database for list of matching entities
        :param validate_request: if True, includes disabled instances in the query and raises a AXApiResourceNotFound error if no fixtures exists satisfying requirement
        :returns: None if no fixtures were found matching any of the requirements"""
        candidate_dict = {}
        for req_name, requirement in requirements.items():
            query = {}
            if validate_request:
                query['deleted'] = False
            else:
                query['available'] = True

            for base_attr, base_val in requirement.items():
                if base_attr == 'attributes':
                    continue
                elif base_attr == 'class':
                    cat = self.fixmgr.get_fixture_class(name=base_val)
                    query['class_id'] = cat.id
                else:
                    query[base_attr] = base_val
            for attr_name, attr_val in requirement.get('attributes', {}).items():
                query['attributes.{}'.format(attr_name)] = attr_val
            candidates = list(self.fixmgr.query_fixture_instances(query))
            if not candidates:
                if validate_request:
                    raise AXApiResourceNotFound("Impossible request: no instances exist satisfying requirement: {}".format(requirement))
                else:
                    logger.debug("Failed to find fixture satisfying: %s", requirement)
                    return None
            candidate_dict[req_name] = candidates
        return candidate_dict

    def _assign_candidates(self, candidate_map):
        """Finds a combination of assignments that will satisfy the mapping of requirements to candidates

        :param candidate_map: mapping of requirement_name to list of fixture candidates json
        :returns: a mapping of the requirement name to the assignment
        """
        logger.debug("Formulating assignments for: %s", list(candidate_map.keys()))
        # Convert the candidate map to a sorted list of tuples of (requirement_name, fixture_id_set).
        # The list is sorted by most restrictive requirement to least restrictive requirement, in order to have
        # a faster assignment algorithm.
        candidate_list = sorted(candidate_map.items(), key=lambda x: len(x[1]))
        candidate_list = [(req_name, set([fix.id for fix in fix_list])) for (req_name, fix_list) in candidate_list]

        fix_id_assignment = self._assign_candidates_helper(candidate_list)
        if fix_id_assignment:
            assignment = {}
            for req_name, fix_id in fix_id_assignment.items():
                assignment[req_name] = self.fixmgr.get_fixture_instance(id=fix_id)
            return assignment
        else:
            logger.warning("Assignment is impossible: %s", candidate_list)
            return None

    def _assign_candidates_helper(self, candidate_list):
        """Internal helper to _assign_candidates to recursively find a working assignment combination"""
        assignments = {}
        req_name, candidates = candidate_list[0][0], list(candidate_list[0][1])
        if len(candidate_list) == 1:
            return {req_name : random.choice(candidates)}
        random.shuffle(candidates)
        for candidate in candidates:
            logger.debug("Attempting assignment: '%s' -> '%s'", req_name, candidate)
            assignments[req_name] = candidate

            # construct new candidate list which excludes current assignment from candidates
            sub_candidate_list = []
            possible_assignment = True
            for _req_name, _cands in candidate_list[1:]:
                new_candidate_set = _cands - {candidate}
                if not new_candidate_set:
                    logger.debug("Assignment of '%s' -> '%s' prevents assignment of '%s'", req_name, candidate, _req_name)
                    possible_assignment = False
                    break
                sub_candidate_list.append((_req_name, new_candidate_set))
            if not possible_assignment:
                continue

            sub_assignments = self._assign_candidates_helper(sub_candidate_list)
            if sub_assignments:
                assignments.update(sub_assignments)
                return assignments
        return None

    def _flatten_assignment(self, assignment):
        """Return a flattened fixture request assignment to be pushed to the notification/assignment channel"""
        flattened = {}
        for ref_name, instance in assignment.items():
            flattened[ref_name] = instance.requestdoc()
        return flattened

    def should_notify(self, fix_req):
        """Tests whether if we should notify the requester about his fixture/volume assignment"""
        if not fix_req.assigned:
            return False
        if self.redis_client_notification.client.exists(fix_req.notification_channel):
            # if there is already a notification, no need to re-notify
            return False
        if not self.fixmgr.volumemgr.check_set_volumes_active(fix_req):
            logger.info("Not all volumes active yet for %s. Skipping channel notification", fix_req)
            return False
        return True

    def _notify_channel(self, fix_req):
        """Notify listener by pushing the assignment to the redis list"""
        fix_names = [f['name'] for f in fix_req.assignment.values()]
        vol_ids = [v['id'] for v in fix_req.vol_assignment.values()]
        logger.info("Notifying %s of assignment: %s, vol_assignment: %s", fix_req.service_id, fix_names, vol_ids)
        self.redis_client_notification.rpush(fix_req.notification_channel, fix_req.json(), expire=3600 * 24 * 10, encoder=json.dumps)

    def process_requests(self):
        """Processes the list of all unassigned fixture requests"""
        with self._processor_lock:
            requests = self.get_fixture_requests()
            logger.info("Processing %s requests", len(requests))
            num_assigned = 0
            num_unassigned = 0
            for fix_req in requests:
                try:
                    if not fix_req.assigned:
                        num_unassigned += 1
                        logger.info("Processing request: %s", fix_req.json())
                        if self._process_request(fix_req):
                            num_assigned += 1
                    else:
                        if self.should_notify(fix_req):
                            self._notify_channel(fix_req)

                except Exception:
                    logger.exception("Failed to process request: %s", fix_req)

            logger.info("Assigned %s/%s requests", num_assigned, num_unassigned)
            return num_assigned

    def _process_request(self, fix_req):
        """Processes a single fixture request. Returns the request if it was successfully assigned, None otherwise

        :param fix_req: a FixtureRequest instance"""
        fixture_assignment = None
        fixture_reserve_ids = []
        volume_assignment = None

        if fix_req.requirements:
            # Fixtures are requested by attributes
            candidate_map = self._find_candidates(fix_req.requirements)
            if not candidate_map:
                return None
            fixture_assignment = self._assign_candidates(candidate_map)
            if not fixture_assignment:
                return None
            logger.debug("Preliminary instance assignment for %s:", fix_req)
            for ref_name, instance in fixture_assignment.items():
                logger.debug("%s:\n%s", ref_name, pretty_json(instance.json()))
                fixture_reserve_ids.append(instance.id)

        if fix_req.vol_requirements:
            # Find and assign volumes are which are requested specifically by name
            volume_assignment = self.fixmgr.volumemgr.find_named_volume_candidates(fix_req)
            if volume_assignment is None:
                return None
            # Create and assign anonymous volume requests
            if not self.fixmgr.volumemgr.provision_anonymous_volumes(fix_req, volume_assignment):
                return None

        # If we get here, it means we have successfully found available fixtures and/or volumes which satisfies
        # the fixture request. The following steps will update the databases with the assignments.
        try:
            if fixture_assignment:
                self.reserve_instances(fix_req, fixture_reserve_ids)
                fix_req.assignment = self._flatten_assignment(fixture_assignment)
            if volume_assignment:
                self.fixmgr.volumemgr.reserve_volumes(fix_req, volume_assignment.values())
                fix_req.vol_assignment = self.fixmgr.volumemgr.flatten_assignment(volume_assignment)

            fix_req.assignment_time = int(time.time() * 1e6)
            self.requestdb.update(fix_req)
            if fixture_assignment:
                self.update_service_object(fix_req)
        except Exception:
            logger.exception("Failed to reserve fixtures")
            # If any problems, release the reservations
            self.release_instances(fix_req, fixture_reserve_ids)
            if volume_assignment:
                self.fixmgr.volumemgr.release_volumes(fix_req, [vol.id for vol in volume_assignment.values()])
            raise

        if self.should_notify(fix_req):
            self._notify_channel(fix_req)
        return fix_req

    def _request_processor(self):
        """Background thread which processes the fixture request queue"""
        while True:
            try:
                if self._events == 0:
                    with self._process_cv:
                        # Wait until next process interval, or we are notified of a change, whichever comes first
                        logger.debug("Waiting for event or process interval")
                        if self._process_cv.wait(timeout=self.process_interval):
                            logger.debug("Notified of change event")
                        else:
                            logger.debug("%s seconds elapsed. Forcing processing", self.process_interval)
                if self._stop:
                    logger.debug("Stop requested. Exiting request processor")
                    return
                with self._process_cv:
                    logger.debug("%s events occurred since last processing time", self._events)
                    self._events = 0
                self.process_requests()
            except Exception:
                logger.exception("Request processor failed")

    def update_service_object(self, fix_req):
        """Updates the service object and adds the instances to its 'fixtures' field. This is best effort"""
        logger.info("Updating service %s with assigned instances", fix_req.root_workflow_id)
        try:
            # NOTE: since request processor is single threaded, it is safe to update the service object
            # without a lock, and we are not worried about concurrent updates with axops since we are
            # only updating a single column (fixtures).
            service = self.axdb_client.get_service(fix_req.root_workflow_id)
            service_fixtures = service.get('fixtures') or {}
            # deserialize the json
            for instance_id, serialized_fixture_doc in service_fixtures.items():
                service_fixtures[instance_id] = json.loads(serialized_fixture_doc)

            for ref_name, assignment in fix_req.assignment.items():
                assignment = copy.deepcopy(assignment)
                instance_id = assignment['id']
                if instance_id not in service_fixtures:
                    service_fixtures[instance_id] = assignment
                if fix_req.requester == common.FIX_REQUESTER_AXWORKFLOWADC:
                    # 'service_ids' is a field added specially for the UI so that it can distinguish which steps utilized
                    # which fixtures. We only want to add this for workflows and not deployments, since service_id means
                    # deployment_id in the context of deployments.
                    service_ids = service_fixtures[instance_id].get('service_ids', [])
                    service_id_dict = {
                        'service_id': fix_req.service_id,
                        'reference_name': ref_name
                    }
                    if not next((sid for sid in service_ids if sid == service_id_dict), None):
                        service_ids.append(service_id_dict)
                        service_fixtures[instance_id]['service_ids'] = service_ids
            logger.info("Updating service object with fixture assignment:\n%s", pretty_json(service_fixtures))
            # serialize the json before storing
            for instance_id, deserialized_fixture_doc in service_fixtures.items():
                service_fixtures[instance_id] = json.dumps(deserialized_fixture_doc)
            payload = {
                'template_name': service['template_name'],
                'fixtures': service_fixtures,
                'ax_update_if_exist': "",
            }
            self.axdb_client.update_service(service['task_id'], payload)
        except Exception:
            logger.exception("Failed to update %s service object with fixture assignment", fix_req.root_workflow_id)

    def start_processor(self):
        """Start the background request processing thread"""
        with self._processor_lock:
            if self._request_processor_thread is None:
                logging.info("Request processor starting")
                self._request_processor_thread = threading.Thread(target=self._request_processor,
                                                                  name="request_processor",
                                                                  daemon=True)
                self._request_processor_thread.start()
            else:
                logging.info("Request processor already started")

    def stop_processor(self):
        """Stop the request processor thread if running"""
        with self._processor_lock:
            if self._request_processor_thread:
                logging.info("Request processor stopping")
                self._stop = True
                self.trigger_processor()
                self._request_processor_thread.join()
                self._request_processor_thread = None
                self._stop = False
                logging.info("Request processor stopped")
            else:
                logging.info("Request processor already stopped")

    def trigger_processor(self):
        """Internal trigger to notify request processor to process the request table"""
        with self._process_cv:
            self._events += 1
            self._process_cv.notify()
