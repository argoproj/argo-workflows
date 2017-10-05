#!/usr/bin/env python3
# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

from ax.util.az_patch import az_patch
az_patch()

import argparse
import base64
import json
import logging
import os
import subprocess
import time
import urllib3
import uuid

from retrying import retry

import boto3
import requests

from ax.cloud import Cloud
from ax.kubernetes.client import KubernetesApiClient, swagger_client, retry_unless
from ax.kubernetes.kubelet import KubeletClient
from ax.kubernetes.pod_status import PodStatus
from ax.meta import AXClusterId, AXClusterDataPath
from ax.platform.exceptions import AXPlatformException, AXVolumeOwnershipException
from ax.platform.stats import post_start_container_event
from ax.platform.routes import ServiceEndpoint
from ax.platform.pod import DIND_CONTAINER_NAME
from ax.platform.pod import ARTIFACTS_CONTAINER_SCRATCH_PATH
from ax.platform.container_specs import AX_DOCKER_GRAPH_STORAGE_THRESHOLD_DEFAULT

from ax.platform.cloudprovider.aws import Route53, Route53HostedZone
from ax.platform.sidecar import PodLogManager
from ax.platform.axmon_main import __version__

logger = logging.getLogger("ax.container_waiter")

WAITER_WAIT_TIMEOUT = 5 * 60

"""
post_start_container_event(
                    uuid.UUID(task.uuid).hex, task.jobname,
                    pod.nodename, pod.nodename,
                    sdef.costid, instance_id,
                    sdef.containers[0].resources.cpu_cores,
                    sdef.containers[0].resources.mem_mib,
                    url_run, url_done
                )
"""

container_log_manager = None
NAMESPACE = "axuser"


def get_service_metadata(pstat):

    # annotations is a dict
    annotations = pstat.metadata.annotations
    assert isinstance(annotations, dict), "Expect annotations to be python dict"
    cost_id = annotations["ax_costid"]
    instance_id = annotations["ax_serviceid"]
    service_env = annotations["AX_SERVICE_ENV"]
    service_env_decoded = json.loads(base64.b64decode(service_env).decode("utf-8"))
    service_context = service_env_decoded.get("container", {}).get("service_context", {})
    root_id = service_context.get("root_workflow_id", "")
    leaf_full_path = service_context.get("leaf_full_path", "")
    logger.info("Info from metadata: cost_id=%s, service_id=%s, root_id=%s, leaf_full_path=%s", cost_id, instance_id,
                root_id, leaf_full_path)
    return cost_id, instance_id, root_id, leaf_full_path


def get_log_urls_for_container(pstat, podname, containername, instance_id):
    assert pstat.metadata.self_link, "Pod status does not have self_link"
    url_run = "{}/log?container={}".format(pstat.metadata.self_link, containername)

    cstats = pstat.status.container_statuses
    docker_id = None
    for cstat in cstats:
        if cstat.name != containername:
            continue
        if cstat.container_id is None:
            # Running: The pod has been bound to a node, and all of the containers have been created.
            # At least one container is still running, or is in the process of starting or restarting.
            raise AXPlatformException(
                "log urls can only be obtained after pod {} has started. Current status of container is {}".format(
                    podname, cstat))
        docker_id = cstat.container_id[len("docker://"):]

    assert docker_id is not None, "Docker ID of created container {} in pod {} was not found".format(containername, podname)

    name_id = AXClusterId().get_cluster_name_id()
    bucket = AXClusterDataPath(name_id).bucket()
    prefix = AXClusterDataPath(name_id).artifact()
    url_done = "/{}/{}/{}/{}.{}.log".format(bucket, prefix, instance_id, containername, docker_id)

    return url_run, url_done


def get_elb_addr(deployment_name):
    s = ServiceEndpoint(deployment_name, namespace=NAMESPACE)
    addrs = s.get_addrs()
    assert len(addrs) == 1, "Expect to see a single entry in the elb addrs array {}".format(addrs)
    return addrs[0]


def add_route53_entry(elb_addr):
    host_mapping = os.environ.get("AX_DEPLOYMENT_HOST_MAPPING", None)
    if host_mapping is None:
        return

    client = boto3.client('route53')
    r53client = Route53(client)
    (host_name, _, domain_name) = host_mapping.partition(".")
    zone = Route53HostedZone(r53client, domain_name)
    zone.create_alias_record(host_name, elb_addr)
    logger.debug("Added a host_mapping {} for elb {}".format(host_mapping, elb_addr))


def post_update_to_axevent(jobname, podname, containername, pstat, node_instance_id):
    logger.debug("Trying to post container update event")
    nodename = pstat.spec.node_name
    uid = pstat.metadata.uid

    cost_id, instance_id, _, _ = get_service_metadata(pstat)
    cpu, mem = PodStatus(pstat).get_resources_for_container(containername)
    url_run, url_done = get_log_urls_for_container(pstat, podname, containername, instance_id)

    deployment_name = os.environ.get("AX_DEPLOYMENT", None)
    elb_addr = None
    if deployment_name:
        elb_addr = get_elb_addr(deployment_name)
        add_route53_entry(elb_addr)

    post_start_container_event(
            uuid.UUID(uid).hex,
            jobname,
            node_instance_id, nodename,
            cost_id, instance_id,
            str(cpu), str(mem),
            url_run, url_done,
            endpoint=elb_addr,
            max_retry=180
    )
    logger.debug("post event done")

    # for now return the service_instance_id as this is used later
    return instance_id


def release_volume_for_dind(ref):
    vol_name = os.environ.get("AX_DOCKER_VOLUME_NAME", None)
    pool_name = os.environ.get("AX_DOCKER_POOL_NAME", None)
    gfs_location = os.environ.get("AX_DOCKER_GRAPH_STORAGE_LOCATION", None)
    gfs_threshold = float(os.environ.get("AX_DOCKER_GRAPH_STORAGE_THRESHOLD", AX_DOCKER_GRAPH_STORAGE_THRESHOLD_DEFAULT))
    if vol_name is not None and pool_name is not None:
        delete_vol = False
        if gfs_location is not None:
            try:
                stat = os.statvfs(gfs_location)

                full_fraction = (stat.f_blocks - stat.f_bfree) / (stat.f_blocks * 1.0)
                logger.debug("Graph storage is at {}% and threshold is {}%".format(full_fraction*100.0, gfs_threshold*100.0))
                if full_fraction >= gfs_threshold:
                    delete_vol = True
            except Exception as e:
                logger.warn("Could not get the size of graph storage at {} due to {}".format(gfs_location, e))

        def retry_on_not_ok(result):
            # this should return False if we do not want to retry
            logger.debug("Request succeeded: {}".format(result.ok))
            return not result.ok

        @retry(retry_on_result=retry_on_not_ok,
               wait_exponential_multiplier=100,
               stop_max_attempt_number=10)
        def axmon_api_call(url):
            logger.debug("Requesting AXMON to return volume {} to pool {}".format(vol_name, pool_name))
            try:
                resp = requests.put(url)
            except AXVolumeOwnershipException as e:
                logger.debug("Volume {} is not owned by {} which is odd since we just finished using it. Ignoring...".format(
                    vol_name, ref
                ))
            return resp

        url = "http://axmon.axsys:8901/v1/axmon/volumepool/{}/{}?current_ref={}".format(pool_name, vol_name, ref)
        if delete_vol:
            url += "&mark=True"

        axmon_api_call(url)

dind_container_id = None


def start_log_collectors(pod_name, pod_status):
    global container_log_manager
    _, service_id, root_id, leaf_full_path = get_service_metadata(pod_status)
    container_log_manager = PodLogManager(pod_name=pod_name,
                                          service_id=service_id,
                                          root_id=root_id,
                                          leaf_full_path=leaf_full_path)
    containers = pod_status.status.container_statuses
    for c in containers:
        if not all([c.name, c.container_id]):
            logger.warning("Container %s (%s) is not ready, skip collecting logs", c.name, c.container_id)
            continue
        cname = c.name
        # container id has format "docker://458e7a52444a802327b0ff795eeffb7684300a637a3b08058438c6660706ae14"
        cid = c.container_id[len("docker://"):]
        container_log_manager.start_log_watcher(cname, cid)


def terminate_log_collectors():
    global container_log_manager
    if isinstance(container_log_manager, PodLogManager):
        try:
            container_log_manager.terminate()
            logger.info("All container log collectors exit cleanly")
        except Exception as e:
            logger.exception("Failed to terminate all log collectors. Error: %s", e)


def wait_for_container(jobname,
                       podname,
                       containername,
                       artifact_scratch_path,
                       out_label):
    # start the waiter but it is possible that the event has passed so
    # poll the status once after the waiter is registered and then go
    # to sleep if the container is still running.

    global dind_container_id

    def get_container_status(s):
        c_status = s.get("containerStatuses", None)
        main_container_status = None
        dind_container_status = None
        docker_ids = {}
        for c in c_status or []:
            name = c.get("name", None)
            if not name:
                continue
            if name == containername:
                main_container_status = c
            elif name == DIND_CONTAINER_NAME:
                dind_container_status = c
            cid = c.get("containerID", None)
            if cid:
                l = len("docker://")
                docker_id = cid[l:]
                logger.debug("Docker ID for {} is {}".format(name, docker_id))
                docker_ids[name] = docker_id

        return main_container_status, dind_container_status, docker_ids

    def get_host_ip():
        """
        Get's the IP address of the host in the cluster.
        """
        k8s = KubernetesApiClient()
        resp = k8s.api.list_node()
        assert len(resp.items) == 1, "Need 1 node in the cluster"
        for n in resp.items:
            for addr in n.status.addresses:
                addr_dict = addr.to_dict()
                if addr_dict['type'] == 'InternalIP':
                    return addr_dict['address']

        return None

    def check_pod_status(pod_status):
        status = pod_status.status
        assert isinstance(status, swagger_client.V1PodStatus), "Expect to see an object of type V1PodStatus"
        status_dict = swagger_client.ApiClient().sanitize_for_serialization(status)
        logger.debug("status_dict=%s", status_dict)

        main_container_status, dind_container_status, docker_ids = get_container_status(status_dict)
        if main_container_status is None:
            if status_dict.get("phase", None) == "Pending":
                logger.debug("Pod still in pending state")
                return False
            else:
                logger.error("bad input %s", status_dict)
                logger.error("Could not find container %s in containerStatuses array", containername)
                return False

        try:
            x = main_container_status["state"]["terminated"]
            logger.debug("Current terminated state object is %s", x)
            k8s_info = {"container_status": {}}
            try:
                k8s_info["pod_ip"] = status.pod_ip
                k8s_info["host_ip"] = status.host_ip
                k8s_info["start_time"] = status.start_time
            except Exception:
                pass
            if x is not None:
                try:
                    k8s_info["container_status"][containername] = x
                except Exception:
                    pass
                try:
                    k8s_info["container_status"][DIND_CONTAINER_NAME] = dind_container_status["state"]["terminated"]
                except Exception:
                    pass

                assert docker_ids, "docker_id should be valid when container terminates"
                with open("/docker_id.txt", "w") as f:
                    f.write(json.dumps(docker_ids))
                with open("/k8s_info.txt", "w") as f:
                    f.write(json.dumps(k8s_info))
                if DIND_CONTAINER_NAME in docker_ids:
                    global dind_container_id
                    dind_container_id = docker_ids[DIND_CONTAINER_NAME]
                return True
            else:
                return False
        except KeyError as ke:
            logger.debug("Expected state of terminated state not observed. Got KerError %s", ke)

        return False

    logger.info("jobname=%s podname=%s containername=%s", jobname, podname, containername)
    node_instance_id = "user-node"
    try:
        node_instance_id = Cloud().meta_data().get_instance_id()
    except Exception:
        pass
    logger.info("Using node instance id %s, namespace %s", node_instance_id, NAMESPACE)

    try:
        kubelet_cli = KubeletClient()
    except Exception as e:
        host_ip = get_host_ip()
        kubelet_cli = KubeletClient(host_ip)

    # have to match with conainer_outer_executor.py
    container_done_flag_postfix = "_ax_container_done_flag"
    poll_container_done_flag_file = "{}/{}/{}".format(artifact_scratch_path, out_label, container_done_flag_postfix)
    service_instance_id = None
    check_file_round = 60 * 2

    count = 0
    posted_event = False
    while True:
        try:
            while True:
                count += 1

                # Kubelet client returns an iterator so we make it a list. As in a certain namespace, pod name
                # is unique, it's safe to always get pods[0]
                pods = [p for p in kubelet_cli.list_namespaced_pods(namespace=NAMESPACE, name=podname)]
                pod_status = pods[0]
                assert isinstance(pod_status, swagger_client.V1Pod), "Expect to see an object of type V1Pod"
                assert pod_status.metadata.name == podname

                # both containers are created so we can assume we have all the knowledge we need for posting URL
                if not posted_event:
                    try:
                        if not jobname.startswith('axworkflowexecutor'):
                            service_instance_id = post_update_to_axevent(jobname, podname, containername, pod_status, node_instance_id)
                        start_log_collectors(pod_name=podname, pod_status=pod_status)
                        posted_event = True
                    except Exception as e:
                        logger.exception("Could not post start event due to %s. Will retry later", e)
                        time.sleep(1)
                        if count % 10 != 0:
                            continue

                done = check_pod_status(pod_status)
                logger.debug("Container %s in [%s][%s] done=%s", containername, jobname, podname, done)
                if done:
                    if not posted_event:
                        try:
                            service_instance_id = post_update_to_axevent(jobname, podname, containername, pod_status, node_instance_id)
                            start_log_collectors(pod_name=podname, pod_status=pod_status)
                        except Exception as e:
                            logger.exception("Could not post start event due to %s.", e)

                    # stop the dind container
                    if dind_container_id:
                        exit_code = subprocess.call(["{}/docker".format(ARTIFACTS_CONTAINER_SCRATCH_PATH), "kill", "-s", "INT", dind_container_id])
                        # TODO: Do docker inspect in a loop and make sure that container dies with clean exit code.
                        # TODO: If exit code is non-zero then ask WFE to ensure that it needs to kill the job controller
                        # TODO: before this pod is terminated
                        logger.debug("Exit code of stopping dind container is {}".format(exit_code))

                        # request axmon to delete volume
                        # sidecar still has this code for backward compatibility for tasks that were started
                        # before docker graph storage used per node vol
                        try:
                            release_volume_for_dind(service_instance_id)
                        except Exception:
                            logger.exception("cannot release_volume_for_dind")
                    return

                for _ in range(1, check_file_round):
                    if os.path.exists(poll_container_done_flag_file):
                        logger.debug("Container %s in [%s][%s] has %s",
                                     containername,
                                     jobname,
                                     podname,
                                     poll_container_done_flag_file)
                        # sleep 1 second to let container status propogate
                        time.sleep(1)
                        break
                    else:
                        time.sleep(2)
                else:
                    # after x min
                    logger.debug("No %s yet, check status again", poll_container_done_flag_file)

        except requests.exceptions.HTTPError as he:
            if "NOT FOUND" in str(he):
                logger.exception("Container %s not found, abort", containername)
                return
            else:
                time.sleep(10)
        except urllib3.exceptions.MaxRetryError:
            logger.exception("Sleep 10 seconds and retry")
            time.sleep(10)
        except Exception as e:
            logger.exception("Container %s in [%s][%s]. Exception type: %s", containername, jobname, podname, type(e))
            time.sleep(10)


if __name__ == "__main__":

    parser = argparse.ArgumentParser(description='waiter')
    parser.add_argument('--version', action='version', version="%(prog)s {}".format(__version__))
    _, args = parser.parse_known_args()

    logging.basicConfig(format="%(asctime)s %(levelname)s %(name)s %(lineno)d %(threadName)s: %(message)s")
    logging.getLogger("ax").setLevel(logging.DEBUG)
    logging.getLogger("ax.kubernetes.kubelet").setLevel(logging.INFO)

    target_cloud = os.environ.get("AX_TARGET_CLOUD", Cloud().own_cloud())
    Cloud().set_target_cloud(target_cloud)

    try:
        wait_for_container(jobname=args[0],
                           podname=args[1],
                           containername=args[2],
                           artifact_scratch_path=args[3],
                           out_label=args[4])
        logger.info("wait_for_container done. Waiting for log collectors to finish their jobs ...")
        terminate_log_collectors()
        logger.info("Container waiter quitting ...")
    except Exception:
        logger.exception("caught exception")
    os._exit(0)
