# -*- coding: utf-8 -*-
#
# Copyright 2015-2016 Applatix, Inc. All rights reserved.
#

"""
Manage the creation of volumes
"""

import ast
import copy
import json
import logging
import random
from threading import Lock
import threading

from ax.aws.meta_data import AWSMetaData
from ax.aws.meta_data import AWSMetaData
from ax.cloud.aws import RawEBSVolume
from ax.kubernetes import swagger_client
from ax.kubernetes.client import KubernetesApiClient, parse_kubernetes_exception
from ax.meta import AXClusterId
from ax.platform.exceptions import AXPlatformException, AXVolumeException, AXVolumeExistsException, AXVolumeOwnershipException
from ax.platform.resources import AXResource
from ax.platform_client.env import AXEnv
from ax.util.ax_random import random_string
from ax.util.callbacks import Callback, ReturnWrapperCond
import boto3
import botocore
from future.utils import with_metaclass
from retrying import retry

from .volume_helpers import job_complete


logger = logging.getLogger(__name__)

# changing this to zero so that existing volumepools delete volumes when those volumes
# are returned after platform upgrade
VOLUMEPOOL_HIGH_THRESHOLD_DEFAULT = 0

DISK_CREATION_WAIT_TIME_SEC = 30

class Volume(object):

    def __init__(self, name, namespace="axuser"):
        self.name = name
        self.namespace = namespace
        self.size = None

        kclient = KubernetesApiClient(use_proxy=True)
        self.client = kclient.api

    def create(self, size, pool_name=None, metadata=None, refs=[], exclusive=False):
        if self._get_from_provider() is None:
            # create a dynamically provisioned ebs volume
            labels = None
            if pool_name is not None:
                labels = {"ax-pool-name": pool_name}

            meta = {
                    "volume.alpha.kubernetes.io/storage-class": "something",
                    "ax_refs": "{}".format(str(refs)),
                    "ax_exclusive": "{}".format(str(exclusive))
            }
            if metadata:
                meta["ax_metadata"] = "{}".format(json.dumps(metadata))

            pvc = swagger_client.V1PersistentVolumeClaim()
            pvc.metadata = Volume.generate_metadata(
                name=self.name,
                annotations=meta,
                labels=labels
            )

            spec = swagger_client.V1PersistentVolumeClaimSpec()
            spec.access_modes = ["ReadWriteOnce"]
            resource = swagger_client.V1ResourceRequirements()
            resource.requests = {"storage": "{}Gi".format(size)}
            spec.resources = resource
            pvc.spec = spec

            self.client.create_namespaced_persistent_volume_claim(pvc, self.namespace)
            logger.debug("Created vol {}".format(json.dumps(pvc.to_dict())))

            # wait for the pv name to be populated
            # self._wait_for_pv_in_provider()

            self.size = size
        else:
            raise AXVolumeExistsException("Volume {} already exists".format(self.name))

    def delete(self):
        state = self._get_from_provider()
        if state is not None:
            refs_str = state.metadata.annotations['ax_refs']
            refs = ast.literal_eval(refs_str)
            if len(refs) >= 1:
                raise AXVolumeException("Cannot remove a volume with refs")
            self._delete_from_provider()
            logger.debug("Deleted volume {}".format(state))

    def found(self):
        if self._get_from_provider():
            return True
        else:
            return False

    def check_in_cloud(self):
        """
        This function checks if the volume exists in the cloud provider
        On error it raises AXPlatformException
        """
        state = self._get_from_provider()
        if state is None:
            raise AXPlatformException("Missing volume for {}".format(self.name))

        # This is due to bug in Kubernetes. See AA-1764
        # we only check on adding ref as we want to minimize the number of boto
        # calls. We only care about disk existing when we are adding a ref to the volume
        self._check_volume_in_cloud_provider(state)

    def add_ref(self, ref, exclusive=False):
        """
        Add a reference to the EBS volume
        Args:
            ref: string
            exclusive: Boolean for exclusive access

        Returns: the array of refs after addition of ref
        """
        state = self._get_from_provider()
        if state is None:
            raise AXPlatformException("Missing volume for {}".format(self.name))

        if state.metadata.annotations.get("ax_deletion", "False") == "True":
            raise AXVolumeException("Cannot add ref to a volume that is marked for deletion")

        # refs is an array of ref
        refs_str = state.metadata.annotations['ax_refs']
        refs = ast.literal_eval(refs_str)

        curr_excl = state.metadata.annotations['ax_exclusive'] == 'True'
        single_ref = len(refs) == 1
        ref_exists = ref in refs

        if len(refs) == 0:
            # trivially add ref
            refs.append(ref)
            self._add_state_and_update(exclusive, refs)
            return refs

        if exclusive:
            if single_ref and ref_exists:
                if not curr_excl:
                    # update to exclusive
                    refs.append(ref)
                    self._add_state_and_update(exclusive, refs)
                return refs
            else:
                raise AXVolumeException("Cannot lock volume {} for ref {} in state [Exclusive {} Refs {}]".format(self.name, ref, curr_excl, refs))
        else:
            if not curr_excl:
                if not ref_exists:
                    refs.append(ref)
                    self._add_state_and_update(exclusive, refs)
                return refs
            else:
                raise AXVolumeException("Cannot add ref {} to volume {} as it is state [Exclusive {} Refs {}".format(ref, self.name, curr_excl, refs))

    def delete_ref(self, ref):
        state = self._get_from_provider()
        if state is None:
            raise AXPlatformException("Missing volume for {}".format(self.name))

        # refs is an array of ref
        refs_str = state.metadata.annotations['ax_refs']
        refs = ast.literal_eval(refs_str)

        ref_exists = ref in refs
        if not ref_exists:
            return refs
        newrefs = [r for r in refs if r != ref]
        self._add_state_and_update(False, newrefs)
        return newrefs

    def mark_for_deletion(self):
        state = self._get_from_provider()
        if state is None:
            raise AXPlatformException("Missing volume for {}".format(self.name))

        s = {
            "metadata": {
                "annotations": {
                    "ax_deletion": "True"
                }
            }
        }
        self._update_in_provider(s)

    def is_marked_for_deletion(self):
        state = self._get_from_provider()
        if state is None:
            raise AXPlatformException("Missing volume for {}".format(self.name))

        return state.metadata.annotations.get("ax_deletion", "False") == "True"

    def _add_state_and_update(self, exclusive, refs):
        s = {
                "metadata": {
                    "annotations": {
                        "ax_refs": "{}".format(str(refs)),
                        "ax_exclusive": "{}".format(str(exclusive))
                    }
                }
        }
        self._update_in_provider(s)

    def _get_from_provider(self):
        try:
            status = self.client.read_namespaced_persistent_volume_claim(self.namespace, self.name)
            assert status.metadata.name == self.name, "Object name {} does not match {}".format(status.metadata.name, self.name)
            return status
        except swagger_client.rest.ApiException as e:
            # if we did not find the file then do not raise exception
            if e.status != 404:
                raise e

        return None

    @parse_kubernetes_exception
    def _update_in_provider(self, state):
        # raise all exceptions for now
        self.client.patch_namespaced_persistent_volume_claim(state, self.namespace, self.name)

    @parse_kubernetes_exception
    def _delete_from_provider(self):
        self.client.delete_namespaced_persistent_volume_claim({}, self.namespace, self.name)

    @staticmethod
    def generate_metadata(name=None, labels=None, annotations=None):
        meta = swagger_client.V1ObjectMeta()
        if name is not None:
            meta.name = name
        if labels is not None:
            meta.labels = labels
        if annotations is not None:
            meta.annotations = annotations
        return meta

    # this may take some time so do slow retries
    @retry(wait_exponential_multiplier=1000,
           stop_max_attempt_number=10)
    def _wait_for_pv_in_provider(self):
        status = self._get_from_provider()
        logger.debug("Waiting for volume {} to be provision in provider: status = {}".format(self.name, status))
        if status is None:
            raise AXPlatformException("Cannot find volume {} in provider".format(self.name))
        if status.spec.volume_name is None or status.spec.volume_name == '':
            raise AXPlatformException("Cloud provider has not provisioned a volume for {}".format(self.name))
        else:
            return

    def _check_volume_in_cloud_provider(self, status):

        def get_pv(pv):
            try:
                response = self.client.read_persistent_volume_status(pv)
                return response
            except swagger_client.rest.ApiException as e:
                if e.status != 404:
                    raise e
            return None

        # The multiplier is an order of magnitude slower as we are doing boto
        # calls and amazon has rate limits and api call limits that we do not
        # want to exceed.
        def is_volume_in_cloud_provider(volume):
            try:
                ec2 = boto3.resource('ec2', region_name=AWSMetaData().get_region())
                vol = ec2.Volume(volume)
                state = vol.state
                logger.debug("The current state of volume {} aws vol {} is {}".format(self.name, volume, state))
                return True
            except botocore.exceptions.ClientError as e:
                code = e.response['ResponseMetadata']['HTTPStatusCode']
                # 400 and 404 are for invalid volume id and volume not found
                if code != 404 and code != 400:
                    raise e
            return False

        if status is None or not status.spec.volume_name:
            raise ValueError("Volume {} is not ready yet in kubernetes. Need to wait a while".format(self.name))

        pv_name = status.spec.volume_name

        pv_obj = get_pv(pv_name)
        if pv_obj is None:
            raise AXPlatformException("Could not get persistent volume info for {} ({})".format(self.name, pv_name))

        vol_id = pv_obj.spec.aws_elastic_block_store.volume_id.split('/')[-1]
        if not is_volume_in_cloud_provider(vol_id):
            raise AXPlatformException("Volume {} does not have underlying volume {} in cloud".format(self.name, vol_id))


class AXVolumeResource(AXResource):
    """
    This is a wrapper for Volumes that are AXResources
    """
    def __init__(self, name, namespace, owner, size_in_gb):
        self.name = name
        self.namespace = namespace
        self.owner = owner
        self.size_in_gb = size_in_gb
        self._meta = None

    @staticmethod
    def create_object_from_info(info):
        name = info["name"]
        namespace = info["application"]
        owner = info["owner"]
        size_in_gb = info["size_gb"]
        return AXVolumeResource(name, namespace, owner, size_in_gb)

    def get_resource_name(self):
        return self.name

    def get_resource_info(self):
        return self._meta

    def create(self):

        self._meta = {
            "name": self.name,
            "application": self.namespace,
            "owner": self.owner,
            "size_gb": self.size_in_gb
        }

        v = Volume(self.name, namespace=self.namespace)
        try:
            v.create(self.size_in_gb, refs=[self.owner], exclusive=True)
        except AXVolumeExistsException as ve:
            logger.debug("Looks like the volume {}/{} exists. Lets ensure we own it".format(self.namespace, self.name))
            try:
                v.add_ref(self.owner, exclusive=True)
            except AXVolumeException as v:
                logger.debug("Ownership test failed due to {}".format(v))
                raise v

    def status(self):
        v = Volume(self.name, namespace=self.namespace)
        return {
            "exists": v.found()
        }

    def delete(self):
        v = Volume(self.name, namespace=self.namespace)
        v.delete_ref(self.owner)
        v.delete()

class AXNamedVolumeResource(AXResource):
    """
    This is a wrapper for NamedVolumes that are AXResources
    """
    def __init__(self, name, resource_id):
        self.name = name
        self.resource_id = resource_id
        self._meta = None

    @staticmethod
    def create_object_from_info(info):
        name = info["name"]
        resource_id = info["resource_id"]
        return AXNamedVolumeResource(name, resource_id)

    def get_resource_name(self):
        return self.name

    def get_resource_info(self):
        return self._meta

    def create(self):
        self._meta = {
            "name": self.name,
            "resource_id": self.resource_id
        }

    def status(self):
        return {
            "exists": "True"
        }

    def delete(self):
        # Named volumes are deleted by users.
        return

# This is a helper decorator used by VolumeManager
def wrap_in_condition(func):
    def condition_wrapper(*args, **kwargs):
        cond = ReturnWrapperCond(value=0)
        # this func has no return value
        func(*args, cond=cond, **kwargs)
        cond.acquire()
        if cond.exception:
            raise cond.exception
        return cond.ret

    return condition_wrapper


class VolumePool(object):
    """
    This class manages pools of volumes
    """
    def __init__(self, name, namespace, size, attributes):
        self.name = name
        self.namespace = namespace
        self.size = size
        self.attributes = attributes
        self._volmap = VolumePool._get_volumes_for_pool_from_provider(name, namespace, size, attributes)
        self._count = len(self._volmap)
        self._soft_limit = VOLUMEPOOL_HIGH_THRESHOLD_DEFAULT

        # set some timers for checking the refs in volumes
        for volname, params in self._volmap.iteritems():
            if params["taken"]:
                ref = params["taken-by"]
                params["timer"] = self._create_timer(volname, ref)

    def get(self, ref):
        """
        Get a volume from the pool.
        Returns: Volume
        """
        # shallow copy is intentional so that params points to
        # original dict. Copy is made so that for loop is sane while
        # entries in self._volmap are removed
        volmap_copy = copy.copy(self._volmap)
        for volname, params in volmap_copy.iteritems():
            if not params["taken"]:
                vol = Volume(volname, namespace=self.namespace)
                try:
                    if not vol.is_marked_for_deletion():
                        vol.add_ref(ref, exclusive=True)
                        params["taken"] = True
                        params["taken-by"] = ref
                        params["timer"] = self._create_timer(volname, ref)
                        return vol.name
                    else:
                        self.remove_vol(volname)
                except AXPlatformException as e:
                    # if add ref failed that means the underlying volume is missing
                    self.remove_vol(volname)

        name = "pool-{}-vol-{}".format(self.name, random_string(5))
        vol = Volume(name, namespace=self.namespace)
        try:
            meta = {
                "namespace": self.namespace,
                "size": self.size,
                "attributes": self.attributes
            }
            vol.create(self.size, pool_name=self.name, metadata=meta, refs=[ref], exclusive=True)
        except AXVolumeExistsException:
            logger.warn("[UNEXPECTED] Volume {} already exists so just add a ref".format(name))
            vol.add_ref(ref, exclusive=True)

        self._volmap[name] = {
            "taken": True,
            "taken-by": ref,
            "timer": self._create_timer(name, ref)
        }
        self._count += 1
        return name

    def put(self, volname, current_ref=None):
        """
        Return a volume back to the pool
        Args:
            volname: volume name string
            current_ref: the ref that is requesting the put
        """
        if volname not in self._volmap:
            # this is possible if the first put in sidecar container causes this volume to be deleted
            # followed by an attempt by task.delete to return this volume to the pool
            logger.debug("Volume {} not in volume map {} for pool {}".format(volname, self._volmap, self.name))
            return

        params = self._volmap[volname]
        ref = params["taken-by"]
        if current_ref and current_ref != ref:
            logger.debug("Volume {} put request received for pool {} with ref {} but we think the ref is held by {}".format(
                volname, self.name, current_ref, ref
            ))
            raise AXVolumeOwnershipException("Cannot put volume {} as requesting ref {} is not the same as held ref {}".format(
                volname, current_ref, ref
            ))
        vol = Volume(volname, namespace=self.namespace)
        try:
            self._cancel_timer(params["timer"])
            vol.delete_ref(ref)
            params["taken"] = False
            params["taken-by"] = ""
            params["timer"] = None
        except AXPlatformException as e:
            # volume was unexpectedly not found in provider
            self.remove_vol(volname)

        marked_for_deletion = vol.is_marked_for_deletion()
        if marked_for_deletion or self._count > self._soft_limit:
            # delete this volume either because it was too full or that the volume
            # pool itself is oversubscribed.
            logger.debug("Deleting volume {} in pool {} (count {}/ threshold {}), marked for deletion: {}".format(
                volname, self.name, self._count, self._soft_limit, marked_for_deletion
            ))
            self.remove_vol(volname)

    def remove_vol(self, name):
        if name not in self._volmap:
            logger.debug("Volume {} not in volumepool {}".format(name, self.name))
            return

        vol = Volume(name, namespace=self.namespace)
        vol.delete()

        self._volmap.pop(name)
        self._count -= 1

    def remove_all_vols(self):
        volmap_copy = copy.copy(self._volmap)
        for vol in volmap_copy:
            self.remove_vol(vol)

    def get_json(self):
        ret = {}
        ret["name"] = self.name
        ret["size"] = "{} GiB".format(self.size)
        ret["namespace"] = self.namespace
        ret["attributes"] = self.attributes
        ret["volumes"] = []
        for v, p in self._volmap.iteritems():
            vol_dict = {}
            vol_dict["name"] = v
            vol_dict["Used"] = p["taken"]
            vol_dict["Refs"] = p["taken-by"]
            ret["volumes"].append(vol_dict)
        return ret

    def list_volume_pool_state(self):
        s = "Volume pool: {} Size {} Namespace {} Attributes {} Count {}\n".format(self.name, self.size, self.namespace, self.attributes, self._count)
        for vol in self._volmap:
            s += "Vol {} State {}\n".format(vol, self._volmap[vol])

        return s

    def list_volumes_in_pool_from_provider(self):
        @parse_kubernetes_exception
        @retry(wait_exponential_multiplier=100,
               stop_max_attempt_number=10)
        def get_vols_from_kubernetes():
            client = KubernetesApiClient(use_proxy=True)
            vols = client.api.list_namespaced_persistent_volume_claim(self.namespace, label_selector="ax-pool-name={}".format(self.name))
            assert isinstance(vols, swagger_client.V1PersistentVolumeClaimList)
            s = ""
            for vol in vols.items or []:
                name = vol.metadata.name
                exclusive = vol.metadata.annotations["ax_exclusive"] == 'True'
                refs_str = vol.metadata.annotations["ax_refs"]
                refs = ast.literal_eval(refs_str)
                pool_meta = json.loads(vol.metadata.annotations["ax_metadata"])
                deletion = vol.metadata.annotations.get("ax_deletion", "False")

                s += "Vol {} Excl {} Refs {} Attributes {} Marked for deletion {}\n".format(name, exclusive, refs, pool_meta, deletion)
            return s

        return get_vols_from_kubernetes()

    def is_ref_valid(self, volume, ref):
        # check if volume exists. If volume does not exists it means that the
        # timer object was incorrectly called for a volume that no longer exists
        if volume not in self._volmap:
            return

        # if the current ref does not match then timer was incorrectly called
        # when the ref has changed
        current_ref = self._volmap[volume]["taken-by"]
        if current_ref == "" or current_ref != ref:
            return

        # check if the job is still active
        logger.debug("Checking if the ref {} is still active for volume {} in pool {}".format(ref, volume, self.name))

        if job_complete(ref):
            # job is complete but we still have an active ref so we need to put the volume back in pool
            logger.debug("Job {} is complete but volume {} has active ref. Giving up ref".format(ref, volume))
            self.put(volume, current_ref=ref)
            return

        logger.debug("Job {} is still running...".format(ref))
        # finally create the timer again. Job is active right now so we may need to check again
        self._volmap[volume]["timer"] = self._create_timer(volume, ref)

    def _create_timer(self, volume, ref):
        # get a random value between 5 and 10 minutes so that all timers are not firing at the same time
        if ref.startswith("applatix.io/deployment"):
            logger.debug("Timers for deployment volumes are not supported yet")
            return
        time_in_seconds = random.randint(300, 600)
        logger.debug("Creating a timer that will fire in {} seconds for volume {} with ref {}".format(time_in_seconds, volume, ref))
        timer = threading.Timer(time_in_seconds, VolumeManager.timer_ref_check, [self.namespace, self.name, volume, ref])
        timer.start()
        return timer

    def _cancel_timer(self, timer):
        if timer is not None:
            logger.debug("Removing volume pool timer as ref has been removed")
            timer.cancel()

    @staticmethod
    def _get_volumes_for_pool_from_provider(pool_name, namespace, size, attributes):

        @parse_kubernetes_exception
        @retry(wait_exponential_multiplier=100,
               stop_max_attempt_number=10)
        def get_vols_from_kubernetes():
            client = KubernetesApiClient(use_proxy=True)
            vols = client.api.list_namespaced_persistent_volume_claim(namespace, label_selector="ax-pool-name={}".format(pool_name))
            assert isinstance(vols, swagger_client.V1PersistentVolumeClaimList)
            ret = {}
            for vol in vols.items or []:
                logger.debug("Processing volume {}".format(json.dumps(vol.to_dict())))
                name = vol.metadata.name
                exclusive = vol.metadata.annotations["ax_exclusive"] == 'True'
                refs_str = vol.metadata.annotations["ax_refs"]
                refs = ast.literal_eval(refs_str)
                pool_meta = json.loads(vol.metadata.annotations["ax_metadata"])

                vol_obj = Volume(name, namespace)
                if len(refs) == 0:
                    vol_obj.delete()
                    continue

                logger.warn("Volume {}/{} not deleted as it has references".format(namespace, name))

                if pool_meta["size"] != size or pool_meta["attributes"] != attributes:
                    # if we have a size of attributes mismatch then do not use this volume if it has no ref
                    # if it has a ref then mark it for deletion upon return to the pool
                    if len(refs) > 0:
                        vol_obj.mark_for_deletion()
                    else:
                        vol_obj.delete()
                        continue

                assert len(refs) <= 1
                ret[name] = {
                    "taken": exclusive,
                    "taken-by": refs[0] if exclusive else [],
                    "timer": None
                }
            return ret

        return get_vols_from_kubernetes()


class VolumeManagerSingleton(type):
    """
    This is a special implementation of singleton that is used for supporting
    singleton volume manager for each namespace
    """
    _instances = {}

    def __call__(cls, namespace="axuser"):
        if namespace not in cls._instances:
            cls._instances[namespace] = super(VolumeManagerSingleton, cls).__call__(namespace)
        return cls._instances[namespace]


class VolumeManager(with_metaclass(VolumeManagerSingleton, object)):
    """
    This class will process all volume related requests in a single thread.
    We do this to ensure that all volume related code consistent information.
    """
    def __init__(self, namespace="axuser"):
        logger.debug("Starting volume manager for {}".format(namespace))
        self._namespace = namespace

        # map of volume pools
        self._pools = {}
        existing_pool_list = VolumeManager._get_pools_from_provider(namespace)
        for poolname, meta in existing_pool_list.iteritems():
            meta_dict = json.loads(meta)
            size = meta_dict["size"]
            attribs = meta_dict["attributes"]
            self._pools[poolname] = VolumePool(poolname, namespace, size, attribs)

        self._cb = Callback()
        self._cb.add_cb(self._handle_volume_event)
        self._cb.start()

        # The following are used for raw EBS volumes.
        def get_region():
            return AWSMetaData().get_region() if AXEnv().is_in_pod() else "us-west-2"
        self.ec2 = boto3.Session().client('ec2', region_name=get_region())
        self.cluster_id = AXClusterId().get_cluster_name_id()
        # This lock is used for synchronizing raw ebs volume creations
        self.raw_disk_lock = Lock()

    @wrap_in_condition
    def create(self, name, size, **kwargs):
        self._cb.post_event("create", name, size=size, **kwargs)

    @wrap_in_condition
    def delete(self, name, mark=False, **kwargs):
        self._cb.post_event("delete", name, mark=mark, **kwargs)

    @wrap_in_condition
    def add_ref(self, name, ref, exclusive=False, **kwargs):
        self._cb.post_event("add_ref", name, ref=ref, exclusive=exclusive, **kwargs)

    @wrap_in_condition
    def delete_ref(self, name, ref, **kwargs):
        self._cb.post_event("delete_ref", name, ref=ref, **kwargs)

    @wrap_in_condition
    def check_vol(self, name, **kwargs):
        self._cb.post_event("check_vol", name, **kwargs)

    @wrap_in_condition
    def create_pool(self, name, size, attributes, **kwargs):
        self._cb.post_event("create_pool", name, size=size, attributes=attributes, **kwargs)

    @wrap_in_condition
    def delete_pool(self, name, **kwargs):
        self._cb.post_event("delete_pool", name, **kwargs)

    @wrap_in_condition
    def get_volume_from_pool(self, name, ref, **kwargs):
        self._cb.post_event("get_from_pool", name, ref=ref, **kwargs)

    def get_from_pool(self, name, ref, **kwargs):

        def retry_exception(exception):
            """
            Return True for retry
            """
            logger.debug("Got exception {}".format(exception))
            if isinstance(exception, AXPlatformException) or isinstance(exception, AssertionError):
                return False
            return True

        @retry(wait_exponential_multiplier=1000,
               stop_max_attempt_number=10,
               retry_on_exception=retry_exception)
        def check_vol_with_retry(volname):
            self.check_vol(volname)

        volname = self.get_volume_from_pool(name, ref, **kwargs)
        check_vol_with_retry(volname)
        return volname

    @wrap_in_condition
    def put_in_pool(self, name, volume_name, current_ref=None, **kwargs):
        self._cb.post_event("put_in_pool", name, volume_name=volume_name, current_ref=current_ref, **kwargs)

    @wrap_in_condition
    def delete_volume_from_pool(self, name, volume_name, **kwargs):
        self._cb.post_event("delete_volume_from_pool", name, volume_name=volume_name, **kwargs)

    @wrap_in_condition
    def list_pools(self, **kwargs):
        self._cb.post_event("list_pools", None, **kwargs)

    @wrap_in_condition
    def ref_check(self, name, volume_name, ref, **kwargs):
        self._cb.post_event("ref_check", name, volume_name=volume_name, ref=ref, **kwargs)

    def create_raw_volume(self, volume_id, vol_opts):
        raw_volume = RawEBSVolume(self.ec2, volume_id, self.cluster_id)
        with self.raw_disk_lock:
            return raw_volume.create(vol_opts)

    def get_raw_volume(self, volume_id):
        raw_volume = RawEBSVolume(self.ec2, volume_id, self.cluster_id)
        with self.raw_disk_lock:
            return raw_volume.query_aws_volume_info()

    def update_raw_volume(self, volume_id, volume_tags):
        raw_volume = RawEBSVolume(self.ec2, volume_id, self.cluster_id)
        with self.raw_disk_lock:
            raw_volume.create_or_update_tags(volume_tags)

    def delete_raw_volume(self, volume_id):
        raw_volume = RawEBSVolume(self.ec2, volume_id, self.cluster_id)
        with self.raw_disk_lock:
            raw_volume.delete()

    def pool_exists(self, pool_name, size=None):
        """
        Helper function to check if a pool already exists
        """
        pools = self.list_pools()
        size_gb = size
        if size:
            size_gb = "{} GiB".format(size)
        for pool in pools:
            if pool["name"] == pool_name:
                if size_gb is not None and size_gb != pool["size"]:
                    return False
                else:
                    return True
        return False

    # only for unit tests
    def __str__(self):
        s = "POOLS\n"
        for _, p in self._pools.iteritems():
            s += p.list_volume_pool_state()
            s += p.list_volumes_in_pool_from_provider()

        return s

    # NOTE: for unit test only
    def get_pools_for_unit_test(self):
        return [p.name for _, p in self._pools.iteritems()]

    def _handle_volume_event(self, operation, name, **kwargs):
        """
        This function is called in the callback thread
        """
        assert "cond" in kwargs, "Use @wrap_in_condition decorator for {}".format(operation)
        cond = kwargs["cond"]
        ret = None
        try:
            if operation == "create":
                assert "size" in kwargs, "Create expects a size"
                Volume(name, namespace=self._namespace).create(kwargs["size"])
            elif operation == "delete":
                assert "mark" in kwargs, "Delete expects a mark"
                mark = kwargs["mark"]
                if mark:
                    Volume(name, namespace=self._namespace).mark_for_deletion()
                else:
                    Volume(name, namespace=self._namespace).delete()
            elif operation == "add_ref":
                assert "ref" in kwargs, "add_ref expects a ref"
                assert "exclusive" in kwargs, "add_ref expects an exclusive"
                ret = Volume(name, namespace=self._namespace).add_ref(kwargs["ref"], kwargs["exclusive"])
            elif operation == "delete_ref":
                assert "ref" in kwargs, "delete_ref expects a ref"
                ret = Volume(name, namespace=self._namespace).delete_ref(kwargs["ref"])
            elif operation == "check_vol":
                Volume(name, namespace=self._namespace).check_in_cloud()
            elif operation == "create_pool":
                assert "size" in kwargs, "create_pool expects a size"
                assert "attributes" in kwargs, "create_pool expects an attribute"
                size = kwargs["size"]
                attributes = kwargs["attributes"]
                if name not in self._pools:
                    self._pools[name] = VolumePool(name, self._namespace, size, attributes)
                else:
                    pool = self._pools[name]
                    if pool.size != size or pool.attributes != attributes or pool.namespace != self._namespace:
                        # recreate volume pool
                        # Note: The timers for active refs from existing volume pool will remain and they can
                        # trigger at any time. When they trigger they will post an event which will be processed
                        # sometime after the creation of the new volumepool object. This object will also create
                        # timers for some of those same refs. The old timer will trigger the ref_check operation
                        # but since that operation goes thorough callbacks, it will find the new volumepool and
                        # object.
                        self._pools[name] = VolumePool(name, self._namespace, size, attributes)

                ret = True
            elif operation == "delete_pool":
                pool = self._pools.get(name, None)
                if pool:
                    pool.remove_all_vols()
                    self._pools.pop(name)
            elif operation == "get_from_pool":
                pool = self._pools[name]
                assert "ref" in kwargs, "get_from_pool expects a ref"
                ref = kwargs["ref"]
                ret = pool.get(ref)
            elif operation == "put_in_pool":
                pool = self._pools[name]
                assert "volume_name" in kwargs, "delete_from_pool expects a volume_name"
                volume_name = kwargs["volume_name"]
                current_ref = kwargs["current_ref"]
                pool.put(volume_name, current_ref=current_ref)
            elif operation == "delete_volume_from_pool":
                assert "volume_name" in kwargs, "delete_volume_from_pool expects a volume_name"
                volume_name = kwargs["volume_name"]
                pool = self._pools[name]
                pool.remove_vol(volume_name)
            elif operation == "list_pools":
                ret = []
                for pname in self._pools:
                    pool_dict = {}
                    pool = self._pools[pname]
                    pool_dict = pool.get_json()
                    ret.append(pool_dict)
            elif operation == "ref_check":
                assert "volume_name" in kwargs, "ref_check expects a volume_name"
                assert "ref" in kwargs, "ref_check expects a ref"
                volume_name = kwargs["volume_name"]
                ref = kwargs["ref"]
                pool = self._pools[name]
                pool.is_ref_valid(volume_name, ref)
            else:
                cond.release()
                assert False, "Unsupported operation {} for volume manager".format(operation)
        except Exception as e:
            cond.exception = e

        cond.ret = ret
        cond.release()

    @staticmethod
    def _get_pools_from_provider(namespace):

        @parse_kubernetes_exception
        @retry(wait_exponential_multiplier=100,
               stop_max_attempt_number=10)
        def get_pools_from_kubernetes():
            client = KubernetesApiClient(use_proxy=True)
            vols = client.api.list_namespaced_persistent_volume_claim(namespace)
            assert isinstance(vols, swagger_client.V1PersistentVolumeClaimList)
            ret = {}
            for vol in vols.items or []:
                logger.debug("Checking volume {}".format(vol.metadata.name))
                if vol.metadata.labels is None:
                    logger.debug("Ignoring volume {} as it does not have labels that match a supported volumepool".format(vol.metadata.name))
                    continue
                pool_name = vol.metadata.labels.get("ax-pool-name", None)
                if pool_name is None:
                    logger.debug("Ignoring volume {} as it does not have labels that match a supported volumepool".format(vol.metadata.name))
                    continue

                if pool_name not in ret:
                    # get the metadata
                    meta = vol.metadata.annotations.get("ax_metadata", None)
                    if meta is None:
                        logger.warn("Ignoring volume {} as it does not have metadata".format(vol.metadata.name))
                        continue
                    ret[pool_name] = meta

            return ret

        return get_pools_from_kubernetes()

    @staticmethod
    def timer_ref_check(namespace, pool_name, volume, ref):
        manager = VolumeManager(namespace)
        manager.ref_check(pool_name, volume, ref)
