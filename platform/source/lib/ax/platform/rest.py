#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

import os
import json
import logging
import time
import traceback
import re
from inspect import isfunction

import boto3
from botocore.exceptions import ClientError as AWSClientError
from flask import Flask, request, make_response
from flask import jsonify as original_jsonify
from gevent import pywsgi
from retrying import retry
from prometheus_client import Summary, Gauge, start_http_server
from threading import RLock
from six import string_types

from argo.services.service import DeploymentService

from ax.aws.meta_data import AWSMetaData
from ax.cloud import Cloud
from ax.exceptions import AXException, AXNotFoundException, AXIllegalArgumentException
from ax.kubernetes import swagger_client
from ax.kubernetes.client import KubernetesApiClient, retry_unless
from ax.kubernetes.swagger_client.rest import ApiException
from ax.meta import AXClusterConfigPath, AXClusterId, AXCustomerId
from ax.platform.application import Application
from ax.platform.ax_asg import AXUserASGManager
from ax.platform.bootstrap import AXBootstrap
from ax.platform.deployments import Deployment
from ax.platform.axmon_main import AXMon
from ax.platform.cluster_config import AXClusterConfig, SpotInstanceOption
from ax.platform.minion_manager import SpotInstanceOptionManager
from ax.platform.volumes import VolumeManager
from ax.platform.exceptions import AXPlatformException
from ax.platform.cloudprovider.aws import Route53, Route53HostedZone
from ax.platform.routes import ExternalRoute
import requests
from ax.cloud.aws.elb import ManagedElb, visibility_to_elb_addr
from werkzeug.exceptions import BadRequest

_app = Flask("AXmon")
axmon = None

# Rlock for counting the max concurrent requests
concurrent_reqs_lock = RLock()
concurrent_reqs = 0
MAX_CONCURRENT_REQS = 100

MINION_MANAGER_HOSTNAME = "http://minion-manager.kube-system"
MINION_MANAGER_PORT = "6000"

kubectl = KubernetesApiClient(use_proxy=True)
cluster_name_id = os.getenv("AX_CLUSTER_NAME_ID", None)
asg_manager = AXUserASGManager(os.getenv("AX_CLUSTER_NAME_ID"),
                               AXClusterConfig().get_region())

# Need a lock to serialize cluster config operation
cfg_lock = RLock()

axmon_api_latency_stats = Summary("axmon_api_latency", "Latency for axmon REST APIs",
                              ["method", "endpoint", "status"])
axmon_api_concurrent_reqs = Gauge("axmon_api_concurrent_reqs", "Concurrent requests in axmon")


def before_request():
    request.start_time = time.time()
    global concurrent_reqs, MAX_CONCURRENT_REQS, concurrent_reqs_lock
    with concurrent_reqs_lock:
        axmon_api_concurrent_reqs.set(concurrent_reqs)
        # Disabling concurrent request logic for now due to findings in AA-3167
        #if concurrent_reqs >= MAX_CONCURRENT_REQS:
        #    return ax_make_response(
        #        original_jsonify(result="too many concurrent requests (max {})".format(MAX_CONCURRENT_REQS)), 429
        #    )
        concurrent_reqs += 1


def after_request(response):
    global concurrent_reqs, concurrent_reqs_lock
    with concurrent_reqs_lock:
        # Disabling concurrent request logic for now due to findings in AA-3167
        #assert concurrent_reqs > 0, "Concurrent reqs is <= 0 ({}) and decrement is called.".format(concurrent_reqs)
        concurrent_reqs -= 1
        axmon_api_concurrent_reqs.set(concurrent_reqs)

    request_latency = time.time() - request.start_time
    # Remove last components on endpoint to aggregate stats.
    # For example,
    # "/v1/axmon/task/axscm-checkout-bf94ca21-15b5-11e7-8933-0a58c0a8820d" is mapped to "/v1/axmon/task"
    endpoint = re.sub(r"(/v1/axmon/[^/]+)/.*", r"\1", request.path)
    axmon_api_latency_stats.labels(request.method, endpoint, response.status_code).observe(request_latency)
    return response


def prometheus_monitor(app, port):
    start_http_server(port, "")


def jsonify(*args, **kwargs):
    return ax_make_response(original_jsonify(*args, **kwargs), 200)


def ax_make_response(rv, status, headers=None):
    new_headers = {"Pragma": "no-cache",
                   "Cache-Control": "no-cache"}

    request_uuid = request.headers.get("X-Request-UUID", None)
    if request_uuid:
        new_headers["X-Request-UUID-Echo"] = request_uuid

    if headers:
        headers.update(new_headers)
    else:
        headers = new_headers

    return make_response(rv, status, headers)


@_app.errorhandler(Exception)
def error_handler(error):
    if isinstance(error, AXException):
        status_code = error.status_code

        error_dict = error.json()
        if status_code != 404:
            _app.logger.exception(error_dict)
        return ax_make_response(original_jsonify(error_dict), status_code)
    data = {"message": str(error), "backtrace": str(traceback.format_exc())}
    if isinstance(error, ValueError) or isinstance(error, KeyError) or isinstance(error, AssertionError):
        _app.logger.exception(str(error))
        status_code = 400
    else:
        _app.logger.exception("Internal error")
        status_code = 400
    return ax_make_response(original_jsonify(data), status_code)


@_app.route('/')
@_app.route("/v1/axmon/help")
def axmon_api_help():
    """
    Print out this help.
    """
    return jsonify(_help_msg)


@_app.route("/v1/axmon/ping")
def axmon_api_ping():
    """
    Ping to check whether server is alive.
    """
    return jsonify({"status": "OK"})


@_app.route("/v1/axmon/version")
def axmon_api_version():
    """
    Return current API version.
    """
    return jsonify({"version": axmon.version})


@_app.route("/v1/axmon/cluster/nodes", methods=["GET"])
def cluster_node_config():
    """
    Get cluster config node info
    :return:

    Cluster node config example:
    {
      "placement":"us-west-2b",
      "minion_type":"m3.large",
      "axsys_nodes":2,
      "region":"us-west-2",
      "axuser_placement":"us-west-2b",
      "max_count":5,
      "axuser_on_demand_nodes":0,
      "node_tiers":"applatix/user",
      "min_count":3,
      "master_type":"m3.medium"
    }
    """
    return jsonify(AXClusterConfig().get_node_config())


@_app.route("/v1/axmon/cluster/spot_instance_config", methods=['PUT'])
def put_spot_instance_config():
    (data,) = _get_optional_arguments('enabled')
    if isinstance(data, bool):
        enabled_str = str(data)
    elif isinstance(data, string_types):
        enabled_str = "True" if data.lower() == "true" else "False"
    else:
        raise ValueError("enabled must be string or boolean")
    payload = {'enabled': enabled_str}

    # Get "spot_instances_option" option
    (option,) = _get_optional_arguments('spot_instances_option')
    if option is not None:
        spotOptionMgr = SpotInstanceOptionManager(
            cluster_name_id, AXClusterConfig().get_region())
        asg_names = spotOptionMgr.option_to_asgs(option)
        asg_option = " ".join(asg_names)

        if option == SpotInstanceOption.NO_SPOT:
            _app.logger.info("ASGS passed in a \"none\". Disabling minion-manager.")
            enabled_str = "False"
            payload['enabled'] = enabled_str

        payload['asgs'] = asg_option

    response = requests.put(MINION_MANAGER_HOSTNAME + ":" + MINION_MANAGER_PORT + "/spot_instance_config", params=payload)
    response.raise_for_status()

    _app.logger.info("Change in Spot instance config: {}".format(enabled_str))
    return jsonify({"status": "ok"})


@_app.route("/v1/axmon/cluster/spot_instance_config", methods=['GET'])
def get_spot_instance_config():
    response = requests.get(MINION_MANAGER_HOSTNAME + ":" + MINION_MANAGER_PORT + "/spot_instance_config")
    response.raise_for_status()
    details = response.json()
    assert "asgs" in details, "No asgs returned by minion-manager"

    spotOptionMgr = SpotInstanceOptionManager(
        cluster_name_id, AXClusterConfig().get_region())
    spot_option = spotOptionMgr.asgs_to_option(details["asgs"].split(" "))

    return jsonify({"status": "ok", "enabled": details["status"], "spot_instances_option": spot_option})

@_app.route("/v1/axmon/portal", methods=['GET'])
def axmon_api_get_portal():
    """
    Get portal connection information

    Returns the portal connection information as a json object
    """
    try:
        portal = {
            "cluster_name_id": os.getenv("AX_CLUSTER_NAME_ID"),
            "customer_id": AXCustomerId().get_customer_id()
        }
        return jsonify(portal)
    except Exception as e:
        raise AXPlatformException("Critical environment variable missing: {}".format(e))


@_app.route("/v1/webhook", methods=['PUT'])
def create_webhook():
    """
    Create a kubernetes service load balancer connecting external traffic to
    axops, with a range of trusted ips
    Data input format:
    {
        "port_spec": [
            {
                "name": "webhook",
                "port": 8443,
                "targetPort": 8087
            }
        ],
        "ip_ranges": ["0.0.0.0/0"]
    }
    :return:
    {
        "ingress": "xxxxxx.us-west-2.elb.amazonaws.com",
        "detail": V1Service
    }
    """
    data = request.get_json()
    port_spec = data.get("port_spec", None)
    ip_ranges = data.get("ip_ranges", ["0.0.0.0/0"])
    if not port_spec:
        raise AXIllegalArgumentException("No port spec provided")
    webhook_svc_name = "axops-webhook"
    srv = swagger_client.V1Service()
    srv.metadata = swagger_client.V1ObjectMeta()
    srv.metadata.name = webhook_svc_name
    srv.metadata.labels = {
        "app": webhook_svc_name,
        "tier": "platform",
        "role": "axcritical"
    }
    spec = swagger_client.V1ServiceSpec()
    spec.selector = {
        'app': "axops-deployment"
    }
    spec.type = "LoadBalancer"
    spec.ports = port_spec
    spec.load_balancer_source_ranges = ip_ranges

    srv.spec = spec

    # Don't have to retry here as creating webhook is a manual process
    # and it is fair to throw proper error message to user and have them
    # retry manually
    need_update = False
    try:
        kubectl.api.create_namespaced_service(body=srv, namespace="axsys")
    except ApiException as ae:
        if ae.status == 409:
            need_update = True
        elif ae.status == 422:
            raise AXIllegalArgumentException("Unable to create webhook due to invalid argument", detail=str(ae))
        else:
            raise AXPlatformException("Unable to create webhook due to Kubernetes internal error", detail=str(ae))
    except Exception as e:
        raise AXPlatformException("Unable to create webhook", detail=str(e))

    if need_update:
        update_body = {
            "spec": {
                "ports": port_spec,
                "load_balancer_source_ranges": ip_ranges
            }
        }
        try:
            kubectl.api.patch_namespaced_service(body=update_body, namespace="axsys", name=webhook_svc_name)
        except Exception as e:
            raise AXPlatformException("Unable to update webhook", detail=str(e))

    trail = 0
    rst = {
        "port_spec": port_spec,
        "ip_ranges": ip_ranges
    }
    while trail < 60:
        time.sleep(3)
        try:
            svc = kubectl.api.read_namespaced_service_status(namespace="axsys", name=webhook_svc_name)
            if svc.status.load_balancer and svc.status.load_balancer.ingress:
                rst["hostname"] = svc.status.load_balancer.ingress[0].hostname
                return jsonify(rst)
        except ApiException:
            pass
        trail += 1
    try:
        kubectl.api.delete_namespaced_service(namespace="axsys", name=webhook_svc_name)
    except ApiException as ae:
        if ae.status != 404:
            raise ae
    raise AXPlatformException("Webhook creation timeout")


@_app.route("/v1/webhook", methods=['GET'])
def describe_webhook():
    @retry(wait_exponential_multiplier=1000,
           stop_max_attempt_number=3)
    def _get_webhook_from_kube_with_retry():
        try:
            return kubectl.api.read_namespaced_service_status(namespace="axsys", name="axops-webhook")
        except ApiException as ae:
            if ae.status == 404:
                return None
            else:
                raise

    webhook = _get_webhook_from_kube_with_retry()
    if not webhook or not webhook.spec:
        raise AXNotFoundException("No webhook found")

    rst = {
        "port_spec": [],
        "ip_ranges": webhook.spec.load_balancer_source_ranges,
        "hostname": webhook.status.load_balancer.ingress[0].hostname
    }

    for p in webhook.spec.ports:
        rst["port_spec"].append({
            "name": p.name,
            "port": p.port,
            "targetPort": int(p.target_port)
        })

    return jsonify(rst)


@_app.route("/v1/webhook", methods=['DELETE'])
def delete_webhook():
    webhook_svc_name = "axops-webhook"
    try:
        kubectl.api.delete_namespaced_service(namespace="axsys", name=webhook_svc_name)
    except ApiException as ae:
        if ae.status != 404:
            raise AXPlatformException("Unable to delete webhook", detail=str(ae))
    return jsonify(result="ok")


# TODO: Move this to aws directory.
def update_cluster_sg_aws():
    """
    Ensure argo cluster is opened to the given trusted_cidrs
    Data input format:
    {
        "trusted_cidrs": ["1.1.1.1/32", "2.2.2.2/32", "3.3.3.3/32"]
    }
    :return:
    """
    data = request.get_json()
    ip_ranges = data.get("trusted_cidrs", None)
    if not ip_ranges:
        return jsonify(trusted_cidrs=ip_ranges)

    if not isinstance(ip_ranges, list):
        raise AXIllegalArgumentException("Trusted CIDRs must be a list")

    @retry_unless(status_code=[404, 422])
    def _do_update_axops(ip):
        spec = {
            "spec": {
                "loadBalancerSourceRanges": ip
            }
        }
        kubectl.api.patch_namespaced_service(spec, name="axops", namespace="axsys")

    with cfg_lock:
        cluster_config = AXClusterConfig(cluster_name_id=cluster_name_id)
        current_trusted_cidrs = cluster_config.get_trusted_cidr()

        # Update node security groups
        cloud_util = AXBootstrap(cluster_name_id=cluster_name_id, region=cluster_config.get_region())
        try:
            cloud_util.modify_node_security_groups(
                old_cidr=current_trusted_cidrs,
                new_cidr=ip_ranges,
                action_name="UserInitiatedTrustedCidrChange"
            )
        except AWSClientError as ace:
            # In case of client error, ensure current CIDRs are reverted back.
            # The only inconsistency could be, any CIDRs that user wants to add
            # and does not trigger client error are added to node security groups
            # which is fine as long as we return proper error message to UI, and
            # leave users to fix and retry.
            # Not catching exception here because any CIDR ranges persisted to
            # cluster config should are guaranteed to be acceptable by cloud
            # provider

            # TODO (harry): not efficient here as it potentially checks CIDRs that are not removed
            cloud_util.modify_node_security_groups(
                old_cidr=[],
                new_cidr=current_trusted_cidrs,
                action_name="EnsureExistingDueToError"
            )

            if "InvalidParameterValue" in str(ace):
                raise AXIllegalArgumentException("InvalidParameterValue: {}".format(str(ace)))
            else:
                raise ace

        # Update axops security groups
        _do_update_axops(ip=ip_ranges)

        # Persist cluster config. We need to do it the last as if any of the previous
        # option fails, we should not show up the updated trusted CIDRs on UI from
        # any subsequent GET call
        cluster_config.set_trusted_cidr(cidrs=ip_ranges)
        cluster_config.save_config()

    return jsonify(trusted_cidrs=ip_ranges)


@_app.route("/v1/axmon/cluster/security_groups", methods=['PUT'])
def update_cluster_sg():
    if Cloud().target_cloud_aws():
        update_cluster_sg_aws()
    elif Cloud().target_cloud_gcp():
        pass


@_app.route("/v1/axmon/cluster/security_groups", methods=['GET'])
def get_cluster_trusted_cidrs():
    """
    Return data format:
    {
        "trusted_cidrs": ["1.1.1.1/32", "2.2.2.2/32", "3.3.3.3/32"]
    }
    :return:
    """
    with cfg_lock:
        cluster_config = AXClusterConfig(cluster_name_id=cluster_name_id)
        cidrs = cluster_config.get_trusted_cidr()
    return jsonify(trusted_cidrs=cidrs)


@_app.route("/v1/axmon/task", methods=['POST'])
def task_create():
    """
    This endpoint is used to create a single step of a workflow
    Payload is the same as service create
    """
    (data, ) = _get_required_arguments('service')
    return jsonify(result=axmon.task_create(data))


@_app.route("/v1/axmon/task/<task_id>", methods=['GET'])
def task_show(task_id):
    """
    Get status of a single task
    """
    return jsonify(result=axmon.task_show(task_id))


@_app.route("/v1/axmon/task/<task_id>", methods=['DELETE'])
def task_delete(task_id):
    """
    Show all or a single task
    """
    force = request.args.get("force", "False")
    delete_pod = request.args.get("delete_pod", "False")
    stop_running_pod_only = request.args.get("stop_running_pod_only", "False")
    if stop_running_pod_only == "True":
        result = axmon.task_stop_running_pod(task_id)
    else:
        result = axmon.task_delete(task_id, force=force == "True")
    return jsonify(result=result)

@_app.route("/v1/axmon/task/<task_id>/logs", methods=['GET'])
def task_logs(task_id):
    """
    Get the log endpoint for a running task
    """
    return jsonify(result=axmon.task_log(task_id))


@_app.route("/v1/axmon/volume/<volume_id>", methods=['POST'])
def volume_create(volume_id):
    manager = VolumeManager()

    # Right now, only AWS volumes of type EBS are supported!
    volume_options = request.get_json()
    assert "storage_provider_name" in volume_options and volume_options["storage_provider_name"].lower() == "ebs", "Only EBS volumes supported currently!"

    if "zone" not in volume_options:
        volume_options["zone"] = AWSMetaData().get_zone()

    resource_id = manager.create_raw_volume(volume_id, volume_options)
    return jsonify(result=resource_id)


@_app.route("/v1/axmon/volume/<volume_id>", methods=['DELETE'])
def volume_delete(volume_id):
    manager = VolumeManager()
    manager.delete_raw_volume(volume_id)
    return jsonify(result="ok")


@_app.route("/v1/axmon/volume/<volume_id>", methods=['GET'])
def volume_get(volume_id):
    manager = VolumeManager()
    volume_attrs = manager.get_raw_volume(volume_id)
    return jsonify(result=volume_attrs)


@_app.route("/v1/axmon/volume/<volume_id>", methods=['PUT'])
def volume_update(volume_id):
    manager = VolumeManager()
    tags = request.get_json()
    volume_tags = {}
    for key, value in tags.iteritems():
        volume_tags['Key'] = key
        volume_tags['Value'] = value
    manager.update_raw_volume(volume_id, [volume_tags])
    return jsonify(result="ok")


@_app.route("/v1/axmon/volumepool", methods=['GET'])
def volumepool_list():
    manager = VolumeManager()
    return jsonify(result=manager.list_pools())


@_app.route("/v1/axmon/volumepool/<pool_name>", methods=['POST'])
def volumepool_create(pool_name):
    manager = VolumeManager()
    (size_str, ) = _get_required_arguments('size')
    (attributes, ) = _get_optional_arguments('attributes')
    size = int(size_str)
    if attributes:
        assert isinstance(attributes, dict), "attributes must be a dictionary"
    manager.create_pool(pool_name, size, attributes)
    return jsonify(result="ok")


@_app.route("/v1/axmon/volumepool/<pool_name>", methods=['GET'])
def volumepool_get_volume(pool_name):
    ref = request.args.get('ref', None)
    if ref is None:
        raise AXIllegalArgumentException("Volumepool get must contain a ref=something query parameter")
    manager = VolumeManager()
    volname = manager.get_from_pool(pool_name, ref)
    return jsonify(result=volname)


@_app.route("/v1/axmon/volumepool/<pool_name>/<volume_name>", methods=['PUT'])
def volumepool_return(pool_name, volume_name):
    manager = VolumeManager()
    mark = request.args.get("mark", "False") == "True"
    ref = request.args.get("current_ref", None)
    if mark:
        # mark volume for deletion and then put it in the pool
        manager.delete(volume_name, mark=mark)
    manager.put_in_pool(pool_name, volume_name, current_ref=ref)
    return jsonify(result="ok")


@_app.route("/v1/axmon/volumepool/<pool_name>/<volume_name>", methods=['DELETE'])
def volumepool_delete_volume(pool_name, volume_name):
    manager = VolumeManager()
    manager.delete_volume_from_pool(pool_name, volume_name)
    return jsonify(result="ok")


@_app.route("/v1/axmon/volumepool/<pool_name>", methods=['DELETE'])
def volumepool_delete(pool_name):
    manager = VolumeManager()
    volname = manager.delete_pool(pool_name)
    return jsonify(result="ok")


@_app.route("/v1/axmon/registry", methods=['PUT'])
def axmon_registry():
    """
    {
    "id": "acc719f9-8202-11e6-8196-02420af40306",
    "url": "docker.foo.com",
    "category": "registry",
    "type": "private_registry",
    "password": "password",
    "username": "username",
    "hostname": "docker.foo.com"
    }

    Returns:
        200 status code if all is good
        401 status code if docker login failed
        404 status code if registry server could not be reached
        500 status code on internal platform error
    """
    (server, username, password) = _get_required_arguments('hostname', 'username', 'password')
    axmon.add_registry(server, username, password)
    return jsonify(result="ok")


@_app.route("/v1/axmon/registry/test", methods=['POST'])
def axmon_registry_test():
    """
    Ideally this needs to be q query param but for now Hong prefers this to be a separate endpoint
    See comment for registry endpoint for usage
    """
    (server, username, password) = _get_required_arguments('hostname', 'username', 'password')
    axmon.add_registry(server, username, password, save=False)
    return jsonify(result="ok")


@_app.route("/v1/axmon/registry/<server>", methods=['DELETE'])
def axmon_registry_delete(server):
    axmon.delete_registry(server)
    return jsonify(result="ok")


@_app.route("/v1/axmon/artifactsbase", methods=['GET'])
def axmon_artifacts_base():
    name_id = AXClusterId().get_cluster_name_id()
    account = AXClusterConfigPath(name_id).bucket()
    cluster_artifacts = AXClusterConfigPath(name_id).artifact()
    return jsonify(result="/{}/{}".format(account, cluster_artifacts))


@_app.route("/v1/axmon/dnsname", methods=['PUT'])
def axmon_set_dnsname():
    """
    """
    (dnsname,) = _get_required_arguments('dnsname')
    axmon.set_dnsname(dnsname)
    return jsonify(result="ok")


@_app.route("/v1/axmon/application", methods=['POST'])
def axmon_application_create():
    """
    Create an application. If the application name already exists then
    this call will do nothing but also not report an exception. To find
    out if the application exists and its status, use the GET method.
    The body of the post must pass a dict with the following format:
    { 'name': 'somename' }
    The name value must be a DNS compatible name
    Returns:
        A json dict on success in the following format
        { 'result': 'ok' }
    """
    (applicationname, ) = _get_required_arguments('name')
    application = Application(applicationname)
    application.create()
    return jsonify(result="ok")


@_app.route("/v1/axmon/application/<applicationname>", methods=['GET'])
def axmon_application_status(applicationname):
    """
    Returns the status of the application
    Args:
        applicationname: the application name
    Returns: A json dict with the following format
        { 'result': { 'component': True/False } }
        where component is namespace, registry and monitor
    """
    application = Application(applicationname)
    status = application.status()
    return jsonify(result=status)


@_app.route("/v1/axmon/application/<applicationname>", methods=['DELETE'])
def axmon_application_delete(applicationname):
    """
    Delete the application. This has an optional query parameter:
    timeout: In seconds.
    Returns: A json dict with the status of the request in the following format
        { 'result': 'ok' }
    """
    timeout = request.args.get('timeout', None)
    if timeout is not None:
        timeout = int(timeout)
    application = Application(applicationname)
    application.delete(timeout=timeout)
    return jsonify(result="ok")


@_app.route("/v1/axmon/application/<applicationname>/deployment", methods=['POST'])
def axmon_deployment_create(applicationname):
    """
    Create a deployment in an application. This is idempotent
    The body of the post is the deployment template template in the 'template' field.
    Returns:
        A json dict on success in the following format
        { 'result': 'ok' }
    """
    data = request.get_json()
    _app.logger.debug("Deployment create for {}".format(json.dumps(data)))
    s = DeploymentService()
    s.parse(data)

    assert applicationname == s.template.application_name, \
        "Application name {} does not match the one in template {}".format(applicationname, s.template.application_name)

    deployment = Deployment(s.template.deployment_name, applicationname)
    deployment.create(s)
    return jsonify(result="ok")


@_app.route("/v1/axmon/application/<applicationname>/deployment/<deploymentname>", methods=['DELETE'])
def axmon_deployment_delete(applicationname, deploymentname):
    """
    Delete a deployment in an application. This is idempotent
    Query Param:
      timeout: In seconds or None for infinity
    Returns:
        A json dict on success in the following format
        { 'result': 'ok' }
    """
    timeout = request.args.get('timeout', None)
    if timeout is not None:
        timeout = int(timeout)
    deployment = Deployment(deploymentname, applicationname)
    deployment.delete(timeout=timeout)
    return jsonify(result="ok")


@_app.route("/v1/axmon/application/<applicationname>/deployment/<deploymentname>", methods=['GET'])
def axmon_deployment_status(applicationname, deploymentname):
    """
    Get status of a deployment in an application. This is idempotent
    Returns:
        A json dict on success in the following format
        { 'result': 'ok' }
    """
    deployment = Deployment(deploymentname, applicationname)
    status = deployment.status()
    return jsonify(result=status)


@_app.route("/v1/axmon/application/<applicationname>/deployment/<deploymentname>/scale", methods=['PUT'])
def axmon_deployment_scale_update(applicationname, deploymentname):
    """
    Change the scale of a deployment. The body should have the following format
    {
        "replicas": integer
    }
    Returns:
        A json dict on success in the following format
        { 'result': 'ok' }
    """
    (replicas,) = _get_required_arguments('replicas')
    deployment = Deployment(deploymentname, applicationname)
    deployment.scale(replicas)
    return jsonify(result="ok")


@_app.route("/v1/axmon/application/<applicationname>/deployment/<deploymentname>/external_route", methods=['POST'])
def axmon_deployment_external_route(applicationname, deploymentname):
    """
    Creates an external route for a deployment in an application. Note this is used to create
    external routes out of sync with service templates and generally not fully supported.
    Data for post is in the following format
    {
        "dns_name": the dns domain name to be used
        "target_port": the port to point to in the deployment
        "whitelist": An list of cidrs
        "visibility": "world" or "organization"
    }
    """
    (dns_name, target_port, whitelist, visibility,) = _get_required_arguments('dns_name', 'target_port', 'whitelist', 'visibility')
    deployment = Deployment(deploymentname, applicationname)
    selector = deployment.get_labels()
    er = ExternalRoute(dns_name, applicationname, selector, target_port, whitelist, visibility)
    elb_addr = visibility_to_elb_addr(visibility)
    return jsonify(result=er.create(elb_addr))


@_app.route("/v1/axmon/application/<applicationname>/deployment/<deploymentname>/external_route/<dns_name>", methods=['DELETE'])
def axmon_deployment_external_route_delete(applicationname, deploymentname, dns_name):
    deployment = Deployment(deploymentname, applicationname)
    selector = deployment.get_labels()
    er = ExternalRoute(dns_name, applicationname, selector, 80)
    er.delete()
    return jsonify(result="ok")


@_app.route("/v1/axmon/domains", methods=['GET'])
def axmon_domains_list():
    """
    Return a list of hosted zones (domains) that the cluster has access to
    Returns:
        { 'result': [list of domains] }
    """
    if Cloud().target_cloud_gcp():
        return jsonify([])
    else:
        r53client = Route53(boto3.client("route53"))
        return jsonify(result=[x.name for x in r53client.list_hosted_zones()])


@_app.route("/v1/axmon/domains/<domainname>", methods=['GET'])
def axmon_domains_domain(domainname):
    """
    Return a list of records for the domain
    Returns:
        { 'result': [list of records for domain] }
    """
    if Cloud().target_cloud_gcp():
        return jsonify([])
    else:
        r53client = Route53(boto3.client("route53"))
        zone = Route53HostedZone(r53client, domainname)
        return jsonify(result=[x for x in zone.list_records()])


@_app.route("/v1/axmon/managed_elb", methods=['POST'])
def axmon_create_managed_elb():
    """
    Post data is of the following format
    {
        name: name of elb (32 chars)
        application: name of application
        deployment: name of deployment
        deployment_selector: {
            'key1': 'value1',
            'key2: 'value2'
        }
        type: internal/external
        ports: [ an array of PortSpec ]

        PortSpec {
            listen_port: listen on port number
            container_port: container port to forward to
            protocol: tcp/ssl/http/https
            certificate: (optional) only needed if type is ssl/https
        }

        }
    }

    Returns:
        { result: The dns name of the managed elb }
    """
    data = request.get_json()
    (name, application, deployment, deployment_selector, type, ports, ) = _get_required_arguments('name', 'application', 'deployment', 'deployment_selector', 'type', 'ports')
    elb = ManagedElb(name)
    dnsname = elb.create(application, deployment, ports, False if type == "external" else True, labels=deployment_selector)
    return jsonify(result=dnsname)

@_app.route("/v1/axmon/managed_elb/<elb_name>", methods=['DELETE'])
def axmon_delete_managed_elb(elb_name):
    elb = ManagedElb(elb_name)
    elb.delete()
    return jsonify(result="ok")


@_app.route("/v1/axmon/managed_elb", methods=['GET'])
def axmon_list_managed_elb():
    return jsonify(result=ManagedElb.list_all())


@_app.route("/v1/debug/operations", methods=['GET'])
def axmon_debug_ops():
    from ax.platform.operations import OperationsManager
    ops = OperationsManager()
    ops.lockstats()
    return jsonify(result="ok")


@_app.route("/v1/config/max_concurrent_reqs", methods=['GET'])
def config_max_concurrent_reqs():
    global concurrent_reqs, MAX_CONCURRENT_REQS
    return jsonify(max_concurrent_reqs=MAX_CONCURRENT_REQS)


@_app.route("/v1/config", methods=['POST'])
def configure_axmon():
    data = request.get_json()
    global MAX_CONCURRENT_REQS
    if 'max_concurrent_reqs' in data:
        new_val = int(data['max_concurrent_reqs'])
        _app.logger.info("Setting max_concurrent_reqs {} -> {}".format(MAX_CONCURRENT_REQS, new_val))
        MAX_CONCURRENT_REQS = new_val

    return jsonify(result="ok")


@_app.route("/v1/axmon/test", methods=['POST'])
def axmon_test():
    """
    This endpoint is used to test axmon

    Post a JSON in the following format
    {
        'testname': 'name',
    }
    """
    data = request.get_json()
    if data is None:
        _app.logger.debug("No data received.")
        return jsonify({})

    _app.logger.debug("Data received is %s", data)
    if "testname" not in data:
        _app.logger.debug("testname needs to be specified")
        return jsonify({})

    return jsonify(axmon.run_test(data["testname"], data))


def _get_required_arguments(*args):
    """Returns a tuple of required param values from a request payload"""
    try:
        data = request.get_json() or {}
        return tuple(map(lambda arg: data[arg], args))
    except BadRequest:
        raise ValueError("Invalid json: {}".format(request.get_data()))
    except KeyError as e:
        raise ValueError("Missing required argument: {}".format(str(e)))


def _get_optional_arguments(*args):
    """Returns a tuple of optional param values from a request payload"""
    try:
        data = request.get_json() or {}
    except BadRequest:
        raise ValueError("Invalid json: {}".format(request.get_data()))
    return tuple(map(lambda arg: data[arg] if arg in data else None, args))


def axmon_rest_start(port):
    """
    Start Flask http server
    """

    global axmon
    axmon = AXMon()

    _app.logger.setLevel(logging.DEBUG)

    # install handlers used for monitoring and other instrumentation
    _app.before_request(before_request)
    _app.after_request(after_request)

    prometheus_monitor(_app, port + 1)
    http_server = pywsgi.WSGIServer(('', port), _app)
    _app.logger.info("AXMon %s started. API server is serving on port: %s", axmon.version, port)
    http_server.start()


# Must be last in this module to make help work.
_all_locals = dict(locals())
_help_msg = {}
for f in _all_locals:
    if isfunction(_all_locals[f]) and f.startswith("axmon_api_"):
        _help_msg[f.replace("axmon_api_", "")] = _all_locals[f].__doc__.splitlines()
