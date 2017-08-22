# -*- coding: utf-8 -*-
#
# Copyright 2015-2016 Applatix, Inc. All rights reserved.
#

"""
Module to provide exec console access and live logs to running containers
"""

import datetime
import json
import logging
import pprint
import re
import select
import socket
import threading
from collections import defaultdict

import boto3
import requests
from geventwebsocket.exceptions import WebSocketError

from ax.exceptions import AXApiResourceNotFound
from ax.kubernetes.client import KubernetesApiClient
from ax.platform.pod import SIDEKICK_WAIT_CONTAINER_NAME, DIND_CONTAINER_NAME

DEFAULT_TERM_SHELL = 'sh'
DEFAULT_TERM_WIDTH = 150
DEFAULT_TERM_HEIGHT = 30

SIDEKICK_CONTAINERS = [SIDEKICK_WAIT_CONTAINER_NAME, 'kubectl-proxy', DIND_CONTAINER_NAME]

logger = logging.getLogger('ax.platform.console')

knclient = KubernetesApiClient()

s3 = boto3.resource('s3')

AXMON_URL = "http://axmon:8901"

def get_pods(**kwargs):
    for pod in knclient.api.list_pod_for_all_namespaces(**kwargs).items:
        start_date = datetime.datetime.strptime(pod.status.start_time, "%Y-%m-%dT%H:%M:%SZ").replace(microsecond=0)
        end_date = datetime.datetime.utcnow().replace(microsecond=0)
        duration = end_date - start_date
        pod_json = {
            'name' : pod.metadata.name,
            'namespace' : pod.metadata.namespace,
            'host_ip' : pod.status.host_ip,
            'pod_ip': pod.status.pod_ip,
            'phase': pod.status.phase,
            'labels' : pod.metadata.labels,
            'age' : str(duration),
            'container_statuses' : [cs.to_dict() for cs in pod.status.container_statuses]
                if pod.status.container_statuses else [],
            'restarts' : 0,
        }
        for cs in pod_json['container_statuses']:
            pod_json['restarts'] += cs['restart_count']
        yield pod_json


def _get_service_ids(task):
    """Recursively determines a list of service ids related to a task"""
    service_ids = []
    if 'fixtures' in task['template']:
        for step in task['template']['fixtures']:
            for _, sub_svc in step.iteritems():
                service_ids.extend(_get_service_ids(sub_svc))

    if 'steps' in task['template']:
        # workflow case
        for step in task['template']['steps']:
            for _, sub_svc in step.iteritems():
                service_ids.extend(_get_service_ids(sub_svc))
    else:
        service_ids.append(task['id'])
    return service_ids


def get_jobs():
    """Returns list of jobs"""
    response = requests.get('http://axops-internal:8085/v1/services?task_only=true&limit=50')
    response.raise_for_status()
    data = response.json()['data']

    # Do not trust axops for running status. If there is a running pod with an
    # associated root_workflow_id, then consider the job as still running
    running_workflows = set()
    running_pods = knclient.api.list_namespaced_pod('axuser', field_selector='status.phase=Running').items
    for pod in running_pods:
        if 'root_workflow_id' in pod.metadata.labels:
            running_workflows.add(pod.metadata.labels['root_workflow_id'])

    jobs = []
    for job_info in data:
        job_id = job_info['id']
        job = {
            'id': job_id,
            'name': job_info['name'],
            'launch_time': job_info['launch_time'],
        }
        job['status'] = 1 if job_id in running_workflows else job_info['status']
        if 'commit' in job_info:
            job['commit'] = job_info['commit']
        job['user'] = job_info['user']
        jobs.append(job)
    return sorted(jobs, key=lambda j: (bool(j['status'] <= 0), -j['launch_time']))

def stop_job(service_id):
    logger.info("Deleting job %s", service_id)
    response = requests.delete('http://axworkflowadc:8911/v1/adc/workflows/{}'.format(service_id))
    response.raise_for_status()
    result = response.json()
    logger.info("Job delete result %s", result)
    return result


def get_volumepools():
    """Retrieves volumes via axmon"""
    response = requests.get('{}/v1/axmon/volumepool'.format(AXMON_URL))
    response.raise_for_status()
    data = response.json()
    return data['result']

def delete_volumepool(pool_name):
    """Delete a volume pool"""
    response = requests.delete('{}/v1/axmon/volumepool/{}'.format(AXMON_URL, pool_name))
    response.raise_for_status()

def delete_volume(pool_name, volume_name):
    """Delete a volume"""
    response = requests.delete('{}/v1/axmon/volumepool/{}/{}'.format(AXMON_URL, pool_name, volume_name))
    response.raise_for_status()

def _add_pod_state(pod_dict, pod):
    """Helper to copy information from k8 pod object to axconsole pod dictionary"""
    main_ctr = None
    pod_dict['containers'] = []
    for ctr_status in pod.status.container_statuses:
        pod_dict['containers'].append(ctr_status.name)
        if ctr_status.name not in SIDEKICK_CONTAINERS:
            main_ctr = ctr_status
    if not main_ctr:
        logger.warning("Unable to determine main container. Assuming first")
        main_ctr = pod.status.container_statuses[0]
    pod_dict['name'] = main_ctr.name
    pod_dict['image'] = main_ctr.image
    if main_ctr.state.running:
        pod_dict['start_time'] = main_ctr.state.running.started_at
    elif main_ctr.state.terminated:
        pod_dict['start_time'] = main_ctr.state.terminated.started_at
        pod_dict['end_time'] = main_ctr.state.terminated.finished_at
        pod_dict['return_code'] = main_ctr.state.terminated.exit_code

def _get_executor_pod_dict(data):
    """Helper to return a pod dictionary for the axworkflow executor"""
    pod_dict = {}
    pod_dict['service_id'] = data['workflow_id']
    pod_dict['containers'] = []

    # NOTE: we only have container information about axworkflowexecutor if it finishes.
    # Otherwise we need to get it from k8.
    label_name = 'axworkflowexecutor-{}'.format(data['workflow_id'])
    executor_pod = find_service_pod(pod_dict['service_id'], namespace='axuser', label_name=label_name, verify_exists=False)
    if executor_pod:
        _add_pod_state(pod_dict, executor_pod)
    elif 'axworkflowexecutor_callback_result' in data:
        logger.debug("No running pod found for executor")
        pod_dict['name'] = data['axworkflowexecutor_callback_result']['name']
        pod_dict['return_code'] = data['axworkflowexecutor_callback_result']['return_code']
        pod_dict['service_id'] = data['axworkflowexecutor_callback_result']['uuid']
        for event in data.get('events', []):
            if event['event_type'] == 'START':
                pod_dict['start_time'] = event['timestamp']
            elif event['event_type'] == 'TERMINATE':
                pod_dict['end_time'] = event['timestamp']
        logs = data['axworkflowexecutor_callback_result'].get('logs', {})
        for ctr_id in logs.keys():
            pod_dict['containers'].append(ctr_id.split('.')[0])
    return pod_dict

def get_job_pods(service_id):
    response = requests.get('http://axworkflowadc:8911/v1/adc/workflows/{}'.format(service_id))
    response.raise_for_status()
    data = response.json()
    results = defaultdict(dict)

    # NOTE: executor pod may have same service_id as a user pod if it is a container and not a workflow
    # therefore we iterate the leaf_results first and add executor after
    executor_pod_dict = _get_executor_pod_dict(data)

    leaf_results = sorted(data['leaf_results'], key=lambda x: x['timestamp'])
    for leaf in leaf_results:
        try:
            leaf_pod_dict = results[leaf['leaf_id']]
            leaf_pod_dict['service_id'] = leaf['leaf_id']
            leaf_pod_dict['leaf_state'] = leaf['result_code'] # prefer last one
            is_fixture = leaf.get('detail', {}).get('launch_type') == 'fixture'
            if leaf['result_code'] == 'LAUNCHED':
                if is_fixture:
                    leaf_pod_dict['name'] = 'Fixtures: {}'.format(leaf['leaf_id'])
                    leaf_pod_dict['fixtures'] = {}
                    for request_name, assignment in leaf['detail']['output_parameters'].items():
                        leaf_pod_dict['fixtures'][request_name] = assignment['name']
                else:
                    leaf_pod_dict['name'] = leaf['detail']['service_template']['name']
                    leaf_pod_dict['leaf_name'] = leaf['detail']['service_template']['service_context']['leaf_name']
                    leaf_pod_dict['image'] = leaf['detail']['service_template']['template']['container']['image']
                leaf_pod_dict['start_time'] = leaf['timestamp']
            elif leaf['result_code'] == 'SUCCEED':
                leaf_pod_dict['end_time'] = leaf['timestamp']
                rc = leaf['detail'].get('container_return_json', {}).get('return_code')
                if rc is not None:
                    leaf_pod_dict['return_code'] = rc
            elif leaf['result_code'] == 'FAILED':
                if 'start_time' not in leaf_pod_dict:
                    leaf_pod_dict['start_time'] = leaf['timestamp']
                if not is_fixture and 'container_return_json' in leaf['detail']:
                    leaf_pod_dict['return_code'] = leaf['detail']['container_return_json']['return_code']
                leaf_pod_dict['end_time'] = leaf['timestamp']
                leaf_pod_dict['failure_reason'] = leaf['detail']['failure_reason']
            elif leaf['result_code'] == 'INTERRUPTED':
                leaf_pod_dict['end_time'] = leaf['timestamp']
                leaf_pod_dict['failure_reason'] = leaf['result_code']
            logs = leaf.get('detail', {}).get('container_return_json', {}).get('logs')
            if logs:
                leaf_pod_dict['containers'] = []
                for ctr_id in logs.keys():
                    leaf_pod_dict['containers'].append(ctr_id.split('.')[0])
        except Exception:
            logger.exception("Unable to parse leaf:\n%s", pprint.pformat(leaf))

    job_pods = results.values()
    for leaf_pod_dict in job_pods:
        if not leaf_pod_dict.get('containers'):
            # If we get here, container is likely running. Check k8 for the pod info
            leaf_pod = find_service_pod(leaf_pod_dict['service_id'], namespace='axuser', label_name=leaf_pod_dict.get('name'), verify_exists=False)
            if leaf_pod:
                _add_pod_state(leaf_pod_dict, leaf_pod)
            else:
                logger.debug("No logs or running pods found for %s, %s", leaf_pod_dict['service_id'], leaf_pod_dict.get('name'))

    job_pods.append(executor_pod_dict)
    return sorted(job_pods, key=lambda x: x.get('start_time'))


def _exec_recv_thread(exec_sock, client_sock):
    """Thread to handle proxying exec websocket stdout/stderr stream to the client websocket"""
    while exec_sock.connected and not client_sock.closed:
        readable, _, _ = select.select([exec_sock.sock], [], [], 1)
        if readable:
            data = exec_sock.recv()
            if data:
                msg = data[1:]
                if msg:
                    client_sock.send(msg, binary=True)
            else:
                logger.debug("Docker host exec socket closed")
    logger.info("Exiting recv thread (exec_sock.connected=%s, client_sock.connected=%s)", exec_sock.connected, not client_sock.closed)


def _exec_send_thread(exec_sock, client_sock):
    """Thread to handle proxying keystrokes from client websocket to the kubernetes exec websocket stdin stream"""
    try:
        while exec_sock.connected and not client_sock.closed:
            readable, _, _ = select.select([client_sock.stream.handler.socket], [], [], 1)
            if readable:
                message = client_sock.receive() # may raise WebSocketError
                if message and exec_sock.connected:
                    data = '\x00' + message
                    written = exec_sock.send(data)
                    if not written:
                        logger.debug("Docker host exec socket closed")
    except WebSocketError as err:
        logger.debug("Client exec websocket error: %s", err)
    logger.info("Exiting send thread (exec_sock.connected=%s, client_sock.connected=%s)", exec_sock.connected, not client_sock.closed)


def _get_pid(exec_sock, client_sock):
    """Parses the pid of this exec session

    When a client disconnects, we need to ensure their exec session and process are cleaned up. With the K8s/Docker
    exec API, unless a client quits the executed process cleanly (e.g. typing 'exit' from bash), the exec'd process
    will live on inside the container indefinitely. The K8s/Docker remote API currently provides no way to kill/clean
    up previous docker exec sessions. See:
     * https://github.com/docker/docker/issues/9098
     * https://github.com/docker/docker/pull/9994
    Our workaround is to modify the actual cmd to echo the pid before running the requested command. Upon client
    websocket close, we will run a subsequent kill command against the recorded pid. See:
     * https://github.com/docker/docker/issues/9098#issuecomment-189743947
    """
    output = ''
    pid = None
    while True:
        data = exec_sock.recv()
        if data:
            msg = data[1:]
            if msg:
                output += msg
                match = re.match(r"^(\d+\r\n)(.*)", output)
                if match:
                    pid = int(match.group(1))
                    output = match.group(2)
                    break
        if len(output) > 10:
            # We got lots of output (10 characters) without seeing a numeric pid.
            # Give up and just emit the output and move on.
            logger.warning("Unable to determine pid")
            break
    client_sock.send(output, binary=True)
    return pid

def find_service_pod(service_id, **kwargs):
    """Finds running using service_instance_id k8s metadata label"""
    return find_labeled_pod("service_instance_id="+service_id, **kwargs)

def find_deployment_pod(deployment_id, **kwargs):
    """Finds running using deployment_id k8s metadata label"""
    return find_labeled_pod("deployment_id="+deployment_id, **kwargs)

def find_labeled_pod(label_selector, namespace=None, pod_name=None, container=None, label_name=None, verify_exists=True):
    """Find the container associated with a kubernetes label"""
    if namespace:
        service_pods = knclient.api.list_namespaced_pod(namespace, label_selector=label_selector).items
    else:
        service_pods = knclient.api.list_pod_for_all_namespaces(label_selector=label_selector).items
    # NOTE: it's possible for there to be more than one pod associated with a service. In other words, the label
    # selector may return multiple pods. This can happen in a couple scenarios:
    # 1) we are in in a crash loop. In this case we chose the latest
    # 2) a "container" service template was submitted (as opposed to a workflow service template). In this scenario,
    #    the axworkflowexecutor will have same service_id as the user's container. We prefer the user's container
    #    unless label_name is explicitly supplied.
    # 3) It is a scaled deployment (single service_id associated with multiple pods), in this case the pod_name is the specifier
    if len(service_pods) > 1:
        errmsg = "Found {} pods associated with {}: {}".format(len(service_pods), label_selector, [p.metadata.name for p in service_pods])
        if label_selector.startswith("deployment_id=") and not pod_name:
            raise Exception(errmsg)
        else:
            logger.warning(errmsg)
    pod = None
    for candidate in service_pods:
        if pod_name and candidate.metadata.name != pod_name:
            continue
        if label_name and candidate.metadata.labels.get('name') != label_name:
            logger.debug("Ignoring %s != %s", candidate.metadata.labels.get('name'), label_name)
            continue
        if container:
            for ctr_status in candidate.status.container_statuses:
                if ctr_status.name == container:
                    pod = candidate
                    break
            else:
                continue
        if pod is None or candidate.metadata.creation_timestamp > pod.metadata.creation_timestamp:
            pod = candidate
    if not pod and verify_exists:
        raise AXApiResourceNotFound("No pod associated with {} found".format(label_selector))
    return pod


def find_main_container(pod):
    """Find the "main" container of a pod (ignores sidecar containers like 'axsidekickwait' and 'kubectl-proxy')"""
    if len(pod.status.container_statuses) > 1:
        logger.warning("%s has %s containers. Determining main container", pod.metadata.name, len(pod.status.container_statuses))
    ctr = None
    for candidate_ctr in pod.status.container_statuses:
        if candidate_ctr.name in SIDEKICK_CONTAINERS:
            continue
        ctr = candidate_ctr
        break
    return ctr


def kill_pod(namespace, pod, container=None):
    """Kill a pod based on service id"""
    pod_obj = knclient.api.read_namespaced_pod(namespace, pod)
    from ax.kubernetes.swagger_client import V1DeleteOptions
    knclient.api.delete_namespaced_pod(V1DeleteOptions(), pod_obj.metadata.namespace, pod_obj.metadata.name)


def service_logs(service_id, container=None):
    pod_obj = find_service_pod(service_id, container=container, verify_exists=False)
    if pod_obj:
        return pod_log_generator(pod_obj, container=container)
    else:
        return s3_log_generator(service_id, container=container)


def deployment_logs(deployment_id, container=None, pod_name=None):
    pod_obj = find_deployment_pod(deployment_id, container=container, pod_name=pod_name, verify_exists=False)
    if pod_obj:
        return pod_log_generator(pod_obj, container=container)
    else:
        return s3_log_generator(deployment_id, container=container)


def pod_logs(namespace, pod, container=None):
    pod_obj = knclient.api.read_namespaced_pod(namespace, pod)
    return pod_log_generator(pod_obj, container=container)


def pod_log_generator(pod, container=None):
    """Return generator of a pod's log output"""
    threading.current_thread().name = pod.metadata.name + '-log'
    if not container:
        container = find_main_container(pod).name
    response = knclient.get_log(pod.metadata.namespace, pod.metadata.name, container=container, follow=True)
    logger.info("Iterating response")
    try:
        for chunk in response.iter_content(chunk_size=None):
            yield chunk
    finally:
        logger.debug("Closing k8 log stream")
        response.connection.close()


def s3_log_generator(service_id, container=None):
    response = requests.get('http://axworkflowadc:8911/v1/adc/workflows/{}'.format(service_id))
    response.raise_for_status()
    data = response.json()
    if service_id == data['workflow_id']:
        logs = data['axworkflowexecutor_callback_result']['logs']
    else:
        leaf_results = data['leaf_results']
        for leaf in leaf_results:
            logs = leaf['detail'].get('container_return_json', {}).get('logs')
            if logs:
                break
        else:
            raise AXApiResourceNotFound("Workflow executor did not report any logs for container")

    for ctr_id, path in logs.items():
        if container:
            if ctr_id.startswith(container + '.'):
                break
        else:
            if not re.match(r'^({})\.'.format('|'.join(SIDEKICK_CONTAINERS)), ctr_id):
                break
    else:
        raise AXApiResourceNotFound("Unable to find log artifact from s3")
    bucketname, path = path.split('/', 1)
    # TODO: support HTTP range option to prevent downloading of entire file
    #bucket = c.lookup()
    #bucket = s3.Bucket(bucketname)
    #s3_client.get_object(Bucket=bucket)
    #s3.Object()
    logger.info("S3 log located in bucket %s path %s", bucketname, path)
    key = s3.Object(bucketname, path).get()
    return json_log_generator(key['Body'])

    
def json_log_generator(streaming_body):
    try:
        partial = ''
        while True:
            chunk = streaming_body.read(8192)
            if not chunk:
                break
            while True:
                try:
                    nl_index = chunk.index('\n')
                except ValueError:
                    break
                tail = chunk[0:nl_index]
                chunk = chunk[nl_index+1:]
                log_json = json.loads(partial + tail)
                yield log_json['log']
                partial = ''
            partial += chunk
        if partial:
            log_json = json.loads(partial)
            yield log_json['log']
    finally:
        logger.debug("Closing s3 log stream")
        streaming_body.close()

def service_exec_start(client_sock, service_id, **kwargs):
    pod = find_service_pod(service_id)
    return pod_exec_handler(client_sock, pod, **kwargs)

def deployment_exec_start(client_sock, deployment_id, pod_name=None, **kwargs):
    pod = find_deployment_pod(deployment_id, pod_name=pod_name)
    return pod_exec_handler(client_sock, pod, **kwargs)

def pod_exec_start(client_sock, namespace, pod, **kwargs):
    pod_obj = knclient.api.read_namespaced_pod(namespace, pod)
    return pod_exec_handler(client_sock, pod_obj, **kwargs)


def pod_exec_handler(client_sock, pod, cmd=None, container=None, term=None, term_width=None, term_height=None):
    """Start an exec session to a pod and proxy stdin/stdout/stderr between the two websockets"""
    threading.current_thread().name = pod.metadata.name + '-exec'
    if not container:
        container = find_main_container(pod).name

    cmd = cmd or DEFAULT_TERM_SHELL
    term_width = term_width or DEFAULT_TERM_WIDTH
    term_height = term_height or DEFAULT_TERM_HEIGHT

    modified_cmd =  [
        'env',
        'TERM=xterm',
        'COLUMNS={}'.format(term_width),
        'LINES={}'.format(term_height),
        'sh',
        '-c',
        'echo $$ ; {}'.format(cmd)
    ]
    exec_sock = knclient.exec_start(pod.metadata.namespace, pod.metadata.name, modified_cmd,
                                    container=container, stdout=True, stderr=True, stdin=True, tty=True)
    logger.info("Created exec socket session to %s. cmd: %s (%sx%s)", pod.metadata.name, cmd, term_width, term_height)
    pid = _get_pid(exec_sock, client_sock)
    logger.info("Pid determined to be %s", pid)

    # Resize the terminal to match the browser's dimensions.
    # The following is leftover code from docker based implementation using exec API. Kubernetes 1.4 will have a 
    # facility to perform resize (which Docker's remote api already provides).
    #  * https://github.com/kubernetes/kubernetes/pull/25273
    # Keep this code around as reference for when we upgrade to 1.4 since code will be similar. Our workaround in 1.3
    # is to export xterm environment variables (e.g. COLUMNS, LINES) beofre issuing the real command
    #try:
    #    axdc._conn.exec_resize(exec_id, height=term_height, width=term_width)
    #except docker.errors.APIError as e:
    #    # This can happen if user supplied a command that immediately completes
    #    # (such as an invalid command), in which case resize operation is invalid.
    #    # Do not raise error here so that the command output is later passed to
    #    # caller through the web socket.
    #    pass
    #except Exception as e:
    #    logger.warning(e)

    recv_thread = threading.Thread(target=_exec_recv_thread, name=pod.metadata.name+'_recv', args=(exec_sock, client_sock))
    send_thread = threading.Thread(target=_exec_send_thread, name=pod.metadata.name+'_send', args=(exec_sock, client_sock))
    recv_thread.start()
    send_thread.start()
    recv_thread.join()
    send_thread.join()
    if not client_sock.closed:
        logger.info("Waiting for client socket to be closed")
        while client_sock.stream:
            readable, _, _ = select.select([client_sock.stream.handler.socket], [], [], 1)
            if readable:
                try:
                    client_sock.receive() # expected to raise WebSocketError on socket close
                except WebSocketError:
                    break
        logger.info("Client socket closed")
    if pid:
        knclient.exec_kill(pod.metadata.namespace, pod.metadata.name, pid, container=container)
    close_socket(exec_sock)
    logger.info("Exec session to %s completed", pod.metadata.name)


def close_socket(sock):
    """Closes a socket idempotently, ignoring any errors"""
    try:
        sock.shutdown(socket.SHUT_RDWR)
    except Exception:
        pass
    try:
        sock.close()
    except Exception:
        pass
