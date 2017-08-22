"""
Fixturemanager component which handles managing of volumes
"""
import copy
import json
import logging
import queue
import threading
import time
import pprint

from contextlib import ExitStack

from ax.devops.axsys.axsys_client import AxsysClient
from ax.devops.prometheus.prometheus_client import PrometheusClient
from ax.exceptions import AXException, AXApiResourceNotFound, AXIllegalOperationException, AXApiInvalidParam, AXTimeoutException
from ax.util.const import SECONDS_PER_MINUTE

from .common import VolumeStatus, DEFAULT_VOLUME_RETRY_INTERVAL, FIX_REQUESTER_AXAMM, FIX_REQUESTER_AXWORKFLOWADC
from .lockmanager import ResourceLockManager
from .volumestructs import Volume, StorageClass
from .util import TimerThread

logger = logging.getLogger(__name__)

# AXMon currently uses a global lock when performing volume creates/deletes, which means all volume operations
# are serialized. Keep this value small until AXMon increases its concurrency.
DEFAULT_VOLUME_WORKERS = 4

class VolumeOperation(object):
    CREATE = 'create'
    DELETE = 'delete'

class QueuePriority(object):
    HIGH = 0
    LOW = 2

def name_to_axrn(name):
    return 'vol:/'+name

def anonymous_volume_axrn(fix_req, ref_name):
    if fix_req.requester == FIX_REQUESTER_AXAMM:
        return 'vol:/anonymous/application:{}/deployment:{}/{}'.format(fix_req.application_name, fix_req.deployment_name, ref_name)
    elif fix_req.requester == FIX_REQUESTER_AXWORKFLOWADC:
        return 'vol:/anonymous/root_workflow_id:{}/service_id:{}/{}'.format(fix_req.root_workflow_id, fix_req.service_id, ref_name)
    raise AXException("Unknown fixture requester: {}".format(fix_req.requester))

_vol_locker = ResourceLockManager('volume')
lock_volume = _vol_locker.lock_resource

class FixVolumeManager(object):
    """
    Volume manager for FixtureManager
    """

    def __init__(self, fixturemgr, volume_workers=None):
        self.cloud_provider = 'aws'
        self.fixmgr = fixturemgr
        self.axsys_client = AxsysClient()
        self.prometheus_client = PrometheusClient()

        # Internal lock for creating new volumes, or changing the axrn of a volume entries in the database.
        # This is required because axdb provides no contraints for uniqueness against the axrn column.
        self._axrn_lock = threading.Lock()

        # queue of volume_ids in which a status change was made and we need to perform some operation (e.g. create/delete/repair)
        self.volume_work_q = queue.PriorityQueue()

        # Number of volume workers to run
        self.num_workers = DEFAULT_VOLUME_WORKERS if volume_workers is None else volume_workers
        # List of worker threads which have been started
        self._volume_worker_threads = []
        # Dictionary of volumes currently being operated on. This dict prevents multiple operators on a single volume
        self._volume_operations = {}
        # Retry worker and interval
        self._retry_worker = None
        self.retry_interval = DEFAULT_VOLUME_RETRY_INTERVAL
        # API timeout for synchronous volume requests (waiting for anonymous volumes to become active)
        self.sync_vol_request_timeout = 10 * 60
        # Thread for periodically updating volume usage
        self._volume_usage_updater = None

    @property
    def axdb_client(self):
        return self.fixmgr.axdb_client

    def get_storage_class_by_name(self, name):
        """Returns a StorageClass object by its name"""
        doc = self.axdb_client.get_storage_class_by_name(name)
        return StorageClass.deserialize_axdb_doc(doc)

    def create_volume(self, volume_dict, internal=False):
        """Create a volume
        :param volume_dict: dictionary containing the parameters for creating the volume
        :param internal: if internal, bypass validation of create fields (enables creation of anonymous volumes)
        """
        if not internal:
            for key in set(volume_dict.keys()) - Volume.create_fields:
                del volume_dict[key]
        resource_id = volume_dict.get('resource_id')
        if resource_id:
            # We allow the ability to import an existing volume which bypasses the creation logic and sets the volume
            # immediately in an 'active' state. To do so, the incoming volume document will supply 'resource_id'
            volume_dict['status'] = VolumeStatus.ACTIVE

        storage_class_name = volume_dict.get('storage_class')
        if not storage_class_name:
            raise AXApiInvalidParam("Storage class unspecified")
        storage_class = self.get_storage_class_by_name(storage_class_name)
        if not storage_class:
            raise AXApiInvalidParam("Storage class '{}' does not exist".format(storage_class_name))

        # Fill in storage class & provider details into the volume attributes
        volume_dict['storage_class_id'] = storage_class.id
        storage_class_params = copy.deepcopy(storage_class.parameters[self.cloud_provider])
        volume_dict['storage_provider'] = storage_class_params.pop('storage_provider_name')
        volume_dict['storage_provider_id'] = storage_class_params.pop('storage_provider_id')
        volume = Volume(volume_dict)
        # override any user-supplied attributes that collide with the storage class' parameters
        volume.attributes.update(storage_class_params)
        with self._axrn_lock:
            existing = self.axdb_client.get_volume_by_axrn(volume.axrn)
            if existing:
                raise AXIllegalOperationException("Cannot create two volumes with same resource name: {}".format(volume.axrn))
            self.axdb_client.create_volume(volume.axdbdoc())
            logger.info("Initialized %s:\n%s", volume, pprint.pformat(volume.json()))
            if volume.status != VolumeStatus.ACTIVE:
                self.volume_work_q.put((QueuePriority.LOW, volume.id))
        return volume

    def get_volume(self, volume_id, verify_exists=True):
        """Retrieve a volume by its id, or None if it does not exist"""
        volume_doc = self.axdb_client.get_volume(volume_id)
        if not volume_doc:
            if verify_exists:
                raise AXApiResourceNotFound("Volume id {} does not exist".format(volume_id))
            else:
                return None
        return Volume.deserialize_axdb_doc(volume_doc)

    def get_volume_by_axrn(self, axrn, verify_exists=True):
        """Retrieve a volume by its axrn, or None if it does not exist"""
        volume_doc = self.axdb_client.get_volume_by_axrn(axrn)
        if not volume_doc:
            if verify_exists:
                raise AXApiResourceNotFound("Volume {} does not exist".format(axrn))
            else:
                return None
        return Volume.deserialize_axdb_doc(volume_doc)

    def update_volume(self, updates):
        """Update a volume. Only name, axrn, enabled, attributes are supported"""
        # drop any fields which we deem immutable by user
        volume_id = updates['id']
        for key in set(updates.keys()) - Volume.mutable_fields:
            del updates[key]

        with lock_volume(volume_id):
            existing_vol = self.get_volume(volume_id)
            axrn_changed = 'name' in updates and updates['name'] != existing_vol.name
            reenabled = 'enabled' in updates and updates['enabled'] and not existing_vol.enabled
            size_changed = 'attributes' in updates and 'size_gb' in updates['attributes'] and int(updates['attributes']['size_gb']) != int(existing_vol.attributes['size_gb'])
            if size_changed:
                # We do not support size updates at this moment since all our volumes are EBS, which would need filesystem expansion.
                # When we have EFS volumes, revsit and check storage provider to see if it can be supported.
                raise AXIllegalOperationException("{} volume resize is not supported".format(existing_vol.storage_provider))
            # This will detect if user supplied any invalid arguments supplied by user. May raise AXApiInvalidParam
            # If the volume is not yet in an active state, or is in the middle of some operation,
            # reject the name change since platform may be in the middle of creating or deleting the volume.
            if axrn_changed:
                if existing_vol.status != VolumeStatus.ACTIVE:
                    raise AXIllegalOperationException("Volumes cannot be renamed while in '{}' status".format(existing_vol.status))
                current_operation = self._volume_operations.get(existing_vol.id)
                if current_operation:
                    raise AXIllegalOperationException("Volume cannot be renamed. Currently performing '{}' operation", current_operation)
                updates['axrn'] = name_to_axrn(updates['name'])
            existing_json = existing_vol.json()
            existing_json.update(updates)
            validated_vol = Volume(existing_json)

            # If axrn changed, we must acquire axrn lock to prevent duplicate axrns.
            if axrn_changed:
                self._axrn_lock.acquire()

            # Apply the update
            try:
                if axrn_changed or size_changed:
                    if self.axdb_client.get_volume_by_axrn(validated_vol.axrn):
                        raise AXIllegalOperationException("Cannot rename volume: resource name {} already in use".format(validated_vol.axrn))
                    platform_updates = {'id': volume_id}
                    if axrn_changed:
                        platform_updates['axrn'] = validated_vol.axrn
                    if size_changed:
                        platform_updates['attributes'] = {'size_gb': validated_vol.attributes['size_gb']}
                    self.axsys_client.update_volume(platform_updates)
                    # NOTE: if we crash here, EBS volume metadata will be inconsistent with AXDB, and we make no attempt to correct metadata state
                    # This is probably okay since we only use axrn in ebs metadata for debugging purposes
                updates['id'] = validated_vol.id
                updates['mtime'] = time.time()
                self.axdb_client.update_volume(Volume.serialize_axdb_doc(updates))
                logger.info("Updated %s fields:\n%s", existing_vol, pprint.pformat(updates))
            finally:
                if axrn_changed:
                    self._axrn_lock.release()

            axdb_voldoc = self.axdb_client.get_volume(volume_id)
            if reenabled:
                # We just re-enabled a volume, which means we should trigger the processor
                self.fixmgr.reqproc.trigger_processor()
            return Volume.deserialize_axdb_doc(axdb_voldoc)

    def get_volumes(self, anonymous=None, deployment_id=None):
        """Return a list of volumes"""
        if deployment_id:
            volumes = []
            fix_req = self.fixmgr.reqproc.requestdb.get(deployment_id, verify_exists=False)
            if fix_req:
                for vol_assignment in fix_req.vol_assignment.values():
                    vol = self.get_volume(vol_assignment['id'], verify_exists=False)
                    if vol:
                        if anonymous is None or vol.anonymous == anonymous:
                            volumes.append(vol)
            return volumes
        else:
            params = {}
            if anonymous is not None:
                params['anonymous'] = anonymous
            volumes = self.axdb_client.get_volumes(params)
            return [Volume.deserialize_axdb_doc(v) for v in volumes]

    def mark_for_deletion(self, volume_id):
        """Mark a volume for deletion"""
        with lock_volume(volume_id):
            volume = self.get_volume(volume_id, verify_exists=False)
            if volume is None:
                return None
            if volume.status != VolumeStatus.DELETING:
                volume.mark_for_deletion()
                update_doc = {'id': volume_id, 'status': volume.status, 'mtime': int(volume.mtime * 1e6), 'axrn': volume.axrn}
                self.axdb_client.update_volume(update_doc)
            else:
                logger.debug("%s already marked for deletion", volume)
            self.volume_work_q.put((QueuePriority.LOW, volume.id))
            return volume

    def _delete_volume(self, volume_id):
        """Delete volume database record. Should only be called once we have confirmed platform has successfully deleted the volume"""
        self.axdb_client.delete_volume(volume_id)
        return volume_id

    def find_named_volume_candidates(self, fix_req, validate_request=False):
        """Finds volumes which satisfies the volume request for named volumes

        :param fix_req: FixtureRequest with volume requirements to check availability
        :param validate_request: if True, raises a AXApiResourceNotFound error if no volumes exists satisfying requirement
        :returns: dictionary of requirement_name to the volume object that can satisfy the request. None if not all named volume requirements could be satisfied
        """
        candidates = {}
        for ref_name, requirements in fix_req.vol_requirements.items():
            axrn = requirements.get('axrn')
            if not axrn:
                # skip anonymous volume requests
                continue
            volume = self.get_volume_by_axrn(axrn, verify_exists=False)
            if not volume:
                if validate_request:
                    raise AXApiResourceNotFound("Impossible request: {} does not exist".format(axrn))
                logger.warning("No volume with axrn %s exists", axrn)
                return None
            if not validate_request and not volume.is_reservable(fix_req.service_id):
                return None
            candidates[ref_name] = volume
        return candidates

    def provision_anonymous_volumes(self, fix_req, assignment):
        """Creates anonymous volumes requested by the fixture request and updates the assignment dictionary with the created/assigned volumes.

        :param volume_requirements: dictionary of requirement_name (in service template) to request dictionary
        :param assignment: assignment dictionary to update with the assignment
        :returns: modified assignment dictionary with the created volumes. None  requirements could be satisfied
        """
        for ref_name, requirements in fix_req.vol_requirements.items():
            if requirements.get('axrn'):
                # skip named volume requests
                continue
            # NOTE: we passed validation already, by virtue of having a FixtureRequest object, so can safely access these fields
            size_gb = requirements['size_gb']
            storage_class_name = requirements['storage_class']
            storage_class = self.get_storage_class_by_name(storage_class_name)
            if not storage_class:
                # This should not happen since we should have already validated the storage class in create_fixture_request
                raise AXApiInvalidParam("Storage class '{}' does not exist".format(storage_class_name))
            axrn = anonymous_volume_axrn(fix_req, ref_name)
            volume = self.get_volume_by_axrn(axrn, verify_exists=False)
            if volume:
                # An existing anonymous volume can happen if:
                # 1) we create an anonymous volume
                # 2) crash before we notify the requestor about the created & assigned anonymous volumes
                # 3) start fixture manager again and process the request
                logger.warning("Processing a fixture request for anonymous volume which already exists:\n%s", pprint.pformat(volume.json()))
                if not volume.has_referrer(fix_req.service_id):
                    # This should theoretically never happen because we mutate the axrn when we delete volumes
                    # so we would not have found the previous anonymous volume during the call to get_volume_by_axrn().
                    raise AXException("Anonymous {} already exists without {} referrer".format(volume, fix_req.service_id))
            else:
                # NOTE: make owner/creator ids the same as submitter?
                create_payload = {
                    "name" : None,
                    "anonymous" : True,
                    "axrn" : axrn,
                    "storage_class" : storage_class.name,
                    "owner" : fix_req.user,
                    "creator" : fix_req.user,
                    "attributes" : {'size_gb': size_gb},
                    "referrers" : [fix_req.referrer()]
                }
                volume = self.create_volume(create_payload, internal=True)
                assignment[ref_name] = volume
        return assignment

    def reserve_volumes(self, fix_req, volumes):
        """
        Updates the database and appends the referrer to volume's referrers

        :param fix_req: FixtureRequest object of the requestor
        :param volumes: list of volume objects of which to append the referrer to
        """
        logger.info("%s reserving volumes: %s", fix_req, [str(v) for v in volumes])
        volume_ids = [v.id for v in volumes]
        reserved_ids = []
        with ExitStack() as stack:
            # Acquire lock on all volumes. Necessary for atomic assignment of multiple volumes
            for volume_id in volume_ids:
                stack.enter_context(lock_volume(volume_id))

            # We have a lock on all volumes, need to perform one more get to ensure
            # nothing was changed from underneath us (since the call to find_volume_candidates, or while acquire locks)
            try:
                for volume_id in volume_ids:
                    volume = self.get_volume(volume_id)
                    if not volume.is_reservable(fix_req.service_id):
                        raise AXException("{} is no longer reservable".format(volume))
                    modified = volume.add_referrer(fix_req.referrer())
                    if modified:
                        self.axdb_client.update_volume({'id': volume.id, 'referrers': json.dumps(volume.referrers), 'atime': int(time.time() * 1e6)})
                        reserved_ids.append(volume.id)
                        logger.info("Successfully reserved %s with referrer: %s", volume, fix_req)
                    else:
                        logger.warning("%s already had reservation on: %s", fix_req, volume)
            except Exception:
                logger.exception("Failed to reserve volumes. Undoing reservations: %s", reserved_ids)
                try:
                    self._release_volumes(fix_req, reserved_ids)
                except Exception:
                    logger.warning("Failed to release partial reservation")
                raise

    def release_volumes(self, fix_req, volume_ids):
        """Releases volume. Removes the referrer from the volume

        :param fix_req: FixtureRequest object of the requestor
        :param volume_ids: list of volume ids of which to remove the referrer from
        """
        logger.info("%s releasing volumes: %s", fix_req, volume_ids)
        if not volume_ids:
            return
        with ExitStack() as stack:
            # Acquire lock on all volumes. Necessary for atomic release of multiple volumes
            for volume_id in volume_ids:
                stack.enter_context(lock_volume(volume_id))
            self._release_volumes(fix_req, volume_ids)

    def _release_volumes(self, fix_req, volume_ids):
        """Internal helper to release volumes. Lock on volumes is assumed"""
        for volume_id in volume_ids:
            assert volume_id in _vol_locker.resource_locks, "_release_volumes called without locking volume {}".format(volume_id)
            volume = self.get_volume(volume_id, verify_exists=False)
            if not volume:
                logger.warning("Volume %s no longer exists", volume_id)
                continue
            volume.remove_referrer(fix_req.service_id)
            volume.atime = int(time.time())
            if volume.anonymous:
                volume.mark_for_deletion()
            update_payload = {
                'id': volume.id,
                'status': volume.status,
                'referrers': json.dumps(volume.referrers),
                'atime': int(volume.atime * 1e6),
                'mtime': int(volume.mtime * 1e6),
                'axrn': volume.axrn,
            }
            self.axdb_client.update_volume(update_payload)
            if volume.anonymous:
                self.volume_work_q.put((QueuePriority.LOW, volume.id))
            logger.debug("%s release of %s successful", fix_req, volume)

    def all_volumes_active(self, fix_req):
        """Returns true if all volumes in the fixture request are active"""
        for assignment in fix_req.vol_assignment.values():
            vol = self.get_volume(assignment['id'])
            if vol.status != VolumeStatus.ACTIVE:
                return False
        return True

    def wait_volumes_active(self, fix_req):
        """Blocks until the volume assignment of the fixture request all reach a status of active.
        Called when making synchronous anonymous volume requests"""
        if not fix_req.vol_requirements:
            return
        volume_ids = [v['id'] for v in fix_req.vol_assignment.values()]
        logger.info("Waiting for volumes to become active: %s", volume_ids)
        stop_time = time.time() + self.sync_vol_request_timeout
        while True:
            if self.check_set_volumes_active(fix_req):
                break
            if time.time() > stop_time:
                raise AXTimeoutException("Timed out waiting for volumes {} to become active".format(volume_ids))
            time.sleep(5)
        logger.info("All volumes active: %s", volume_ids)

    def check_set_volumes_active(self, fix_req):
        """Check all volumes of the fixture request to see if they are all in active state.
        Updates the fixture request with resource IDs, once they are available"""
        if not fix_req.vol_requirements:
            return True
        volume_ids = [v['id'] for v in fix_req.vol_assignment.values()]
        for volume_id in volume_ids:
            vol = self.get_volume(volume_id, verify_exists=False)
            if not vol:
                # fixture request might become deleted while waiting for volume to be created,
                # which would in turn put volume in deleting state, or be gone entirely.
                raise AXIllegalOperationException("Volume {} deleted while waiting for active status".format(volume_id))
            elif vol.status == VolumeStatus.DELETING:
                raise AXIllegalOperationException("{} marked for deletion while waiting for active status".format(vol))
            elif vol.status == VolumeStatus.ACTIVE:
                assignment = next((assignment for assignment in fix_req.vol_assignment.values() if assignment['id'] == vol.id), None)
                if not assignment.get('resource_id'):
                    assignment['resource_id'] = vol.resource_id
                    self.fixmgr.reqproc.requestdb.update(fix_req)
            else:
                return False
        return True

    def flatten_assignment(self, volume_assignment):
        """Helper to translate a mapping of requirement name to volumes to a payload acceptable for axamm/workflowexecutor"""
        assignment = {}
        for ref_name, volume in volume_assignment.items():
            assignment[ref_name] = volume.axmondoc()
        return assignment

    def usage_updater(self):
        """Updates the usage information for each 'in-use' volume."""
        try:
            # 1. Query AXDB for in-use volumes.
            all_volumes = self.get_volumes()
            volumes_in_use = []
            for v in all_volumes:
                if v.referrers is not None and len(v.referrers) > 0:
                    volumes_in_use.append(v)

            if len(volumes_in_use) == 0:
                logger.info("No volumes being used.")
                return

            logger.info("Resource ids to check: %s", volumes_in_use)

            # 2. Query Prometheus for volume usage info.
            all_volume_json = self.prometheus_client.get_all_volume_free()
            resource_id_to_free_bytes = {}
            for result in all_volume_json["data"]["result"]:
                mountpoint_resource_id = result["metric"]["mountpoint"].split("/")[-1]
                resource_id_to_free_bytes[mountpoint_resource_id] = int(result["value"][1])

            logger.info("Free bytes info from Prometheus: %s", resource_id_to_free_bytes)

            # 3. For each of the volumes in (1) update the DB.
            for volume in volumes_in_use:
                free_bytes = resource_id_to_free_bytes.get(volume.resource_id, -1)
                # Ignore volumes whose information wasn't found in prometheus.
                if free_bytes == -1:
                    continue

                new_attrs = volume.attributes
                new_attrs['free_bytes'] = free_bytes

                updates = {}
                updates['id'] = volume.id
                updates['attributes'] = new_attrs
                self.update_volume(updates)
        except Exception as e:
            logger.warning("Failure while updating volume usage info: " + str(e))

    def start_workers(self):
        """Start volume workers"""
        if len(self._volume_worker_threads) == 0:
            logger.info("Starting %d volume workers", self.num_workers)
            self._retry_worker = TimerThread(self.retry_interval, target=self.retry_volume_operations, name="VolumeRetryWorker")
            self._retry_worker.start()
            for i in range(self.num_workers):
                wkr_thread = threading.Thread(target=self._volume_worker, name="VolumeWorker-{}".format(i+1))
                wkr_thread.start()
                self._volume_worker_threads.append(wkr_thread)
        else:
            logger.warning("Workers already started. Ignoring start_workers call")

        if self._volume_usage_updater is None:
            self._volume_usage_updater = TimerThread(5 * SECONDS_PER_MINUTE,
                                                     target=self.usage_updater,
                                                     name="VolumeUsageUpdater")
            self._volume_usage_updater.start()
        else:
            logger.warning("Volume usage updater thread already started.")

    def stop_workers(self):
        """Stop volume workers"""
        if self._retry_worker:
            self._retry_worker.stopped.set()
            self._retry_worker = None
            logger.info("Retry worker stopped")
        for _ in range(len(self._volume_worker_threads)):
            self.volume_work_q.put((QueuePriority.HIGH, None))
        for wkr_thread in self._volume_worker_threads:
            wkr_thread.join()
        self._volume_worker_threads = []

        if self._volume_usage_updater:
            self._volume_usage_updater.stopped.set()
            self._volume_usage_updater = None
            logger.info("Volume usage updater stopped")

    def _volume_worker(self):
        """Thread target which will pull work off the volume work queue and operate on volumes according to desired state"""
        logger.info("Volume worker starting")
        while True:
            (_, volume_id) = self.volume_work_q.get()
            try:
                if volume_id is None:
                    # Indicates a stop_workers call was made
                    break
                logger.info("Received operation request on %s", volume_id)
                self._operate_volume(volume_id)
            except Exception:
                logger.error("Failed to operate on volume %s", volume_id)
            finally:
                self.volume_work_q.task_done()
        logger.info("Volume worker stopped")

    def _operate_volume(self, volume_id):
        """Method which does the actual work of bringing a volume to the desired state (create, delete)"""
        with lock_volume(volume_id):
            current_operation = self._volume_operations.get(volume_id)
            if current_operation:
                # Volume is currently already performing an operation. This can happen if a second request to operate
                # on a volume came in while the first request is still being operated on (e.g. user deletes a volume
                # while the volume is still in the middle of creation)
                logger.warning("Volume %s currently performing operation: %s. Skipping operate request", volume_id, current_operation)
                # NOTE: Since we skip this, we rely on a periodic fall back mechanism (retry_volume_operations) which will scan all volumes that
                # are not in the desired state (we need this anyways for fixturemanager startup/recovery/restart, and for retrying failed operations)
                return
            volume = self.get_volume(volume_id, verify_exists=False)
            if volume is None:
                logger.info("Volume %s no longer exists. No operations to perform", volume_id)
                return

            if volume.status == VolumeStatus.INIT:
                volume.status = VolumeStatus.CREATING
                self.axdb_client.update_volume({'id': volume_id, 'status': VolumeStatus.CREATING, 'mtime': int(time.time() * 1e6)})
                self._volume_operations[volume_id] = VolumeOperation.CREATE
            elif volume.status == VolumeStatus.CREATING:
                self._volume_operations[volume_id] = VolumeOperation.CREATE
            elif volume.status == VolumeStatus.DELETING:
                self._volume_operations[volume_id] = VolumeOperation.DELETE
            else:
                logger.info("%s currently in %s status. No operations to perform", volume, volume.status)
                return

        error = None
        try:
            operation = self._volume_operations[volume_id]
            logger.info("Performing '%s' operation on %s:\n%s", operation, volume, pprint.pformat(volume.json()))
            if operation == VolumeOperation.CREATE:
                if volume.status_detail:
                    logger.warning("Re-operating on volume in '%s' state. Previous create may have failed", VolumeStatus.CREATING)
                resource_id = self.axsys_client.create_volume(volume.axmondoc())
                logger.info("%s successfully created as resouce_id: %s", volume, resource_id)
                with lock_volume(volume_id):
                    # Check if status changed from underneath us (e.g. user marked volume for deletion while we were creating it)
                    curr_volume = self.get_volume(volume.id)
                    if curr_volume.status == VolumeStatus.CREATING:
                        self.axdb_client.update_volume({'id': volume.id, 'status': VolumeStatus.ACTIVE, 'status_detail': None, 'resource_id': resource_id, 'mtime': int(time.time() * 1e6)})
                        self.fixmgr.reqproc.trigger_processor()
                    else:
                        logger.warning("%s was marked for deletion during create. Requeuing for re-operate", curr_volume)
                        self.volume_work_q.put((QueuePriority.LOW, volume.id))
            elif operation == VolumeOperation.DELETE:
                self.axsys_client.delete_volume(volume_id)
                logger.info("%s successfully deleted", volume)
                with lock_volume(volume_id):
                    self._delete_volume(volume_id)
            else:
                raise AXException("Unknown volume operation on {}: {}".format(volume, operation))
            # NOTE: axnotification logic will go here
        except Exception as err:
            logger.exception("%s failed operation: %s", volume, operation)
            if isinstance(err, AXException):
                error = err
            else:
                error = AXException(err)
        finally:
            with lock_volume(volume_id):
                del self._volume_operations[volume_id]
                if error:
                    self.axdb_client.update_volume({'id': volume.id, 'status_detail': error.json(), 'mtime': int(time.time() * 1e6)})

    def retry_volume_operations(self):
        """Re-queues volume_ids into the volume work queue which need to be created, or deleted.
        This method is invoked at periodic intervals to correct any volume state
        It also corrects any volume referrers inconsistency states (e.g. volume has referrer of a request that no longer exists)"""
        try:
            active_service_ids = set([r.service_id for r in self.fixmgr.reqproc.requestdb.items()])
            reoperate_ids = set()
            for vol in self.get_volumes():
                if vol.status in [VolumeStatus.INIT, VolumeStatus.CREATING, VolumeStatus.DELETING]:
                    if vol.id not in self._volume_operations:
                        # Volume is in a transition state, but is not being worked on. Indicates
                        # we need to retry the volume operation.
                        reoperate_ids.add(vol.id)

                # Check if volume is being referred to by deleted fixture requests
                deleted_service_ids = [r['service_id'] for r in vol.referrers if r['service_id'] not in active_service_ids]
                if deleted_service_ids:
                    logger.warning("%s is being referred to by a deleted service_id(s): %s", vol, deleted_service_ids)
                    try:
                        with lock_volume(vol.id, timeout=5):
                            for service_id in deleted_service_ids:
                                vol.remove_referrer(service_id)
                            self.axdb_client.update_volume({'id': vol.id, 'referrers': vol.referrers})
                            logger.info("Successfully removed deleted referrers %s from %s", deleted_service_ids, vol)
                    except AXTimeoutException:
                        logger.warning("Could not correct referrers of %s: volume is busy", vol)

            logger.debug("%s volumes need reprocessing: %s", len(reoperate_ids), reoperate_ids)
            for volume_id in reoperate_ids:
                self.volume_work_q.put((QueuePriority.LOW, volume_id))
        except Exception:
            logger.exception("Retry worker failed")
