#!/usr/bin/env python

"""
Deal with incorrect volume mount issues.
"""

from ax.kubernetes.client import KubernetesApiClient
import logging
from ax.kubernetes.ax_kube_event_logger import AXKubeEventLogger
try:
    import Queue as Q
except ImportError:
    import queue as Q
import sys
import time
import threading
from dateutil import parser
import requests
import boto3
import json
import subprocess
from parse import parse

from retrying import retry
from ax.aws.meta_data import AWSMetaData
from ax.kubernetes.swagger_client.rest import ApiException

# Logic for detecting and dealing with erronous mounts:
#  1. Find events which match the following signature:

# {'object': {'count': 1, 'kind': 'Event', 'firstTimestamp': '2016-11-02T16:55:52-07:00', 'lastTimestamp': \
# '2016-11-02T16:51:51-07:00', 'apiVersion': 'v1', 'source': {'component': 'controller-manager'}, \
# 'reason': 'FailedMount', 'involvedObject': {'kind': 'Pod', 'name': 'shri-test-pod', 'namespace': 'axsys', \
# 'apiVersion': 'v1', 'resourceVersion': '801513', 'uid': '6f79e0f8-a11d-11e6-ba5b-02a4598d5bf9'}, \
# 'message': 'Failed to attach volume "pvc-76ae8ce2-9d6e-11e6-ba5b-02a4598d5bf9" on node "ip-172-20-0-215.us-west-2.compute.internal" \
# with: Error attaching EBS volume: VolumeInUse: vol-06c31d843c4914169 is already attached to an instance\n\tstatus code: 400,
# request id: ', 'type': 'Warning', 'metadata': {'name': 'shri-test-pod.14834970224b2247', 'namespace': 'axsys', \
# 'resourceVersion': '8342', 'creationTimestamp': '2016-11-02T16:57:28Z', \
# 'selfLink': '/api/v1/namespaces/axsys/events/shri-test-pod.14834970224b2247', 'uid': '700223a9-a11d-11e6-ba5b-02a4598d5bf9'}}, 'type': 'ADDED'}

#  2. Check (i) if the count of this message is > 2 and (ii) if that error has been hitting for certain time (10 minutes).
#  3. Get the name of the PVC and the volume id.
#  4. Put the PVC into a Queue. A separate thread called 'failed_pvc_q_processor' operates on this queue.
#  5. failed_pvc_q_processor picks this PVC entry from the Queue.
#  6. It checks if there is one and only one pod waiting on this PVC and whether that POD is in "Pending" state.
#  7. If there is only one POD, it proceeds to "detect_and_unmount" the PVC.
#  8. Since, this code itself runs as a POD on each node, it is required to detect whether that node is the one where the PVC is incorrectly mounted.
#  9. This is done by a combination of boto APIs and instance metadata.
# 10. If the instance IDs match, the volume is unmounted and detached from the instance.

logging.basicConfig(format="%(asctime)s %(levelname)s %(name)s %(threadName)s: %(message)s", stream=sys.stdout)
logger = logging.getLogger(__name__)
logger.setLevel(logging.DEBUG)

def get_result_length(parse_result):
    if not parse_result:
        return 0

    count = 0
    for r in parse_result:
        count = count + 1
    return count

class VolumeMountsFixer():
    def __init__(self):
        self.failed_pvc_q = Q.Queue()
        self.client = KubernetesApiClient()

        aws_metadata = AWSMetaData()
        self.my_instance_id = aws_metadata.get_instance_id()
        self.my_region = aws_metadata.get_region()

    class FailedMountInfo():
    # This class has basic information about the failed mount. The PersistentVolume id and the AWS volume instance id.
        def __init__(self, pv_id, vol_id):
            self._pv_id = pv_id
            self._vol_id = vol_id

    def get_local_device_name(self, instance_device_name):
        """
        Converts the AWS device name (sda, sdb, etc.) to the one used inside the instance (xvda, xvdb, etc.).
        :param instance_device_name: The AWS device name.
        """
        return instance_device_name.replace("/dev/sd", "/dev/xvd")

    @retry(wait_exponential_multiplier=100, stop_max_attempt_number=10)
    def detach_volume_from_instance(self, volume, vol_id, instance_id):
        """
        Detach the volume from the instance.
        """
        detach_response = volume.detach_from_instance(DryRun=False, InstanceId=instance_id,
                                                      Device=volume.attachments[0]['Device'], Force=False)
        assert detach_response is not None, "No response when detaching volume from instance"
        logger.info("Detached volume %s from instance %s", vol_id, instance_id)

    def detect_and_unmount(self, failed_mount_info):
        """
        Detect whether the given failed mount is on the current AWS instance and if so, unmount it.
        :param failed_mount_info: The FailedMountInfo object to detect and unmount.
        """
        vol_id = failed_mount_info._vol_id
        ec2 = boto3.resource(service_name='ec2', region_name=self.my_region)
        volume = ec2.Volume(vol_id)
        if len(volume.attachments) == 1:
            if volume.attachments[0]['InstanceId'] == self.my_instance_id:
                device_name = self.get_local_device_name(volume.attachments[0]['Device'])

                # Unmount the volume.
                logger.info("Unmounting %s from self", device_name)

                # The volume may be mounted on multiple mount points on the host. Evidently, if the volume is
                # mounted on N mount points, umount needs to be called N times for the volume to be
                # detachable.
                retries = 10
                while (retries > 0):
                    try:
                        subprocess.check_call(["nsenter", "-t", "1", "--mnt", "umount", device_name])
                        retries = retries - 1
                    except subprocess.CalledProcessError as ce:
                        if ce.returncode & 32: #Returned when a device is already unmounted
                            logger.info("Device %s already unmounted", device_name)
                            break
                        else:
                            logger.error("Failed to unmount %s: %s", device_name, ce)
                            return

                self.detach_volume_from_instance(volume, vol_id, self.my_instance_id)
                kube_event_logger = AXKubeEventLogger(namespace="kube-system", client=self.client)
                kube_event_logger.log_event("ax-volume-mounts-fixer", "Unmounted and detached: " + vol_id + \
                                            "(" + device_name + ") from instance " + self.my_instance_id)
            else:
                logger.info("Volume %s not attached to me %s", vol_id, self.my_instance_id)
        return

    @retry(wait_exponential_multiplier=100, stop_max_attempt_number=10)
    def get_pods_waiting(self, failed_mount_info):
        """
        Given a FailedMountInfo object, find the number of pods waiting on that failed PV (with retries).
        :param failed_mount_info: The FailedMountInfo object to process.
        """
        pvc_details = self.client.api.read_persistent_volume(failed_mount_info._pv_id)
        pvc_name = pvc_details.spec.claim_ref.name

        num_pods_waiting = 0
        for pod in self.client.api.list_pod_for_all_namespaces().items:
            for v in pod.spec.volumes or []:
                if v.persistent_volume_claim and v.persistent_volume_claim.claim_name == pvc_name:
                    logger.info("pvc %s is used by POD %s (%s)", v.persistent_volume_claim.claim_name, pod.metadata.name, pod.status.phase)

                    # Check to see if there are any pods using this volume. If a pod is waiting to get this volume, it is in the
                    # Pending state. If a currently running pod or a pod that has just begun/ended is using the volume, it will
                    # be in Running or Unknown state.
                    # Pods in the Successful and Failed state are safe to ignore. These are pods that used the volume but have
                    # now completed their execution.
                    if pod.status.phase == "Pending":
                        num_pods_waiting = num_pods_waiting + 1
                    elif pod.status.phase in ("Running", "Unknown"):
                        logger.info("Not unmounting. Pod %s is in %s state", pod.metadata.name, pod.status.phase)
                        return -1
        return num_pods_waiting

    def process_failed_pvc(self, failed_mount_info):
        """
        Given a FailedMountInfo object, checks whether there is one and only one POD waiting on it and if so,
        unmounts it.
        :param failed_mount_info: The FailedMountInfo object to process.
        """
        num_pods_waiting = self.get_pods_waiting(failed_mount_info)
        # Iff there is only a single pod waiting on the PV, unmount it.
        if num_pods_waiting == 1:
            self.detect_and_unmount(failed_mount_info)
        else:
            logger.debug("Failed volume not unmounted. Pods waiting %d", num_pods_waiting)

    def failed_pvc_q_processor(self):
        """
        This method is run in a separate thread and processes one failed mount at a time.
        """
        logger.info("Started failed pvc mount processor ...")
        while (True):
            failed_mount_info = self.failed_pvc_q.get()
            try:
                self.process_failed_pvc(failed_mount_info)
            except Exception as e:
                logger.error("Failed while processing failed mounts %s: %s. Putting it back in the queue", failed_mount_info._pv_id, e)
                self.failed_pvc_q.put(failed_mount_info)
            finally:
                self.failed_pvc_q.task_done()
        return

    def process_event_from_kubelet(self, event, error_message):
        """
        This method processes error messages logged from Kubelet.
        """
        format = 'Unable to mount volumes for pod "{}": timeout expired waiting for volumes to attach/mount for pod "{}"/"{}". list of unattached/unmounted volumes=[{}]'
        result = parse(format, error_message)
        if get_result_length(result) != 4:
            logger.error("Failed to parse error message %s", error_message)
            return

        pod_name = result[1]
        pod_namespace = result[2]
        namespace_in_event = event['object']['involvedObject']['namespace']
        assert namespace_in_event == pod_namespace, "Namespace in event (" + namespace_in_event + ") != namespace in error message (" + pod_namespace + ")"

        # Split the failed volume names list
        failed_volume_names = result[3].split(" ")
        logger.info("Failed pvcs to fix: %s", failed_volume_names)
        logger.info("Checking pvcs for pod: %s in namespace %s", pod_name, pod_namespace)
        pod = self.client.api.read_namespaced_pod(pod_namespace, pod_name)
        assert pod is not None, "Failed to find pod " + pod_name
        for v in pod.spec.volumes or []:
            if v.persistent_volume_claim:
                if v.name in failed_volume_names:
                    pvc = self.client.api.read_namespaced_persistent_volume_claim(
                        pod_namespace, v.persistent_volume_claim.claim_name)
                    pv = self.client.api.read_persistent_volume(pvc.spec.volume_name)
                    aws_volume_id = pv.spec.aws_elastic_block_store.volume_id
                    # The aws_volume_id above is of the format aws://<availability_zone>/volume_id.
                    # The remainder of the volume-mounts-fixer code only needs the volume_id.
                    aws_volume_id = aws_volume_id.split("/")[-1]
                    self.failed_pvc_q.put(self.FailedMountInfo(pvc.spec.volume_name, aws_volume_id))
                    logger.info("Found failure for pvc %s with AWS volume id %s", pvc.spec.volume_name, aws_volume_id)

        logger.info("Finished processing error message from kubelet")

    def process_event_from_controller_manager(self, event, error_message):
        """
        This method processes error messages logged from kube-controller-manager.
        """
        # With both checks above passed, add the pv to the Queue for further processing.
        format = 'Failed to attach volume "{}" on node "{}" with: Error attaching EBS volume: VolumeInUse: {} is {}'
        result = parse(format, error_message)
        if get_result_length(result) != 4:
            logger.error("Failed to parse error message %s", error_message)
            return

        pv_id = result[0]
        vol_id = result[2]

        # Just to be sure, confirm that the pv is actually used by the POD.
        pod_name = event['object']['involvedObject']['name']
        pod_namespace = event['object']['involvedObject']['namespace']

        try:
            pod = self.client.api.read_namespaced_pod(pod_namespace, pod_name)
            pv_found = False
            for v in pod.spec.volumes or []:
                if v.persistent_volume_claim:
                    pv = self.client.api.read_namespaced_persistent_volume_claim(
                        pod_namespace, v.persistent_volume_claim.claim_name).spec.volume_name
                    if pv == pv_id:
                        pv_found = True
                        break

            if not pv_found:
                logger.error("PV %s not found in POD %s", pv_id, pod_name)
                return
        except ApiException as ae:
            if ae.status == 404:
                logger.error("POD %s not found", pod_name)
                return
            raise ae

        self.failed_pvc_q.put(self.FailedMountInfo(pv_id, vol_id))
        logger.info("Found mount failure for pv %s", pv_id)

    def do_watch(self):
        # Don't specify any namespace. This will watch all events across all namespaces.
        for event in self.client.watch(item="events", field_selector="reason=FailedMount"):
            reason = event['object']['reason']
            if reason != 'FailedMount':
                logger.error("Found non-failed-mount event: %s", reason)
                continue

            error_message = None
            if 'object' in event and 'message' in event['object']:
                error_message = event['object']['message']

            source = None
            if 'source' in event['object'] and 'component' in event['object']['source']:
                source = event['object']['source']['component']
            logger.info("Found FailedMount event from %s: %s", source, error_message)
            if source == 'controller-manager' or 'kubelet' in source:
                try:
                    # If the event hasn't happened more than once, ignore!
                    if int(event['object']['count']) < 2:
                        logger.info("Ignoring message since count < 2")
                        continue

                    # If failed-mount has happened multiple times, check if enough time has passed. This is to avoid
                    # taking action too early.
                    lastSeen = int(parser.parse(event['object']['lastTimestamp']).strftime('%s'))
                    firstSeen = int(parser.parse(event['object']['firstTimestamp']).strftime('%s'))
                    if lastSeen - firstSeen < 600:  # 10 minutes
                        logger.info("Ignoring event since it's not 10 mins")
                        continue

                    if source == 'controller-manager':
                        self.process_event_from_controller_manager(event, error_message)
                    else:
                        self.process_event_from_kubelet(event, error_message)
                except ValueError as v:
                    pass

    def watch_events(self):
        """
        Watch all events currently happening in the Kubernetes cluster. Specifically, this looks for events that
        report failed disk mounts. The PersistentVolume and volume id are extracted from this message and put into
        a queue for further processing.
        """
        logging.info("Started failed pvc mount watcher...")

        while(True):
            try:
                self.do_watch()
            except Exception as e:
                logger.exception("Exception while watching events: %s", e)
                time.sleep(10)

if __name__ == "__main__":
    vmf = VolumeMountsFixer()
    logger.info("Starting volume-mounts-fixer. Instance id: %s, Region: %s", vmf.my_instance_id, vmf.my_region)
    t = threading.Thread(name="failed_pvc_q_processor", target=vmf.failed_pvc_q_processor)
    t.daemon = True
    t.start()

    vmf.watch_events()
