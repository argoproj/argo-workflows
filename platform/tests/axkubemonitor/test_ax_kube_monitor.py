# -*- coding: utf-8 -*-
#
# Copyright 2015-2016 Applatix, Inc. All rights reserved.
#

import logging
import os
from pprint import pformat
import pytest
import random

from ax.kubernetes.ax_kube_dict import KubeApiObjKind, KubeKind, KubeKindToV1KubeSwaggerObject
from ax.kubernetes.client import retry_unless
from ax.kubernetes.swagger_client import *
from ax.platform.ax_monitor_helper import KubeObjStatusCode, KubeObjWaiter
from retrying import retry
import yaml


DEFAULT_POD_CREATION_TIMEOUT = 360
DEFAULT_PVC_CREATION_TIMEOUT = 600
DEFAULT_PV_DELETE_TIMEOUT = 360
DEFAULT_SVC_CREATION_TIMEOUT = 300
TEST_NAMESPACE = "default"
KUBE_NAMESPACE = "kube-system"

PWD = os.path.dirname(__file__)
logger = logging.getLogger(__name__)
logger.setLevel(logging.INFO)


def wait_for_pod_validator(status):
    return status["phase"] == "Running" or status["phase"] == "Succeeded"


def wait_for_pvc_validator(status):
    return status["phase"] == "Bound"


def pv_release_validator(status):
    return status["phase"] == "Released"


def wait_for_svc_lb_validator(status):
    return bool(
        status["loadBalancer"] and
        status["loadBalancer"]["ingress"] and
        len(status["loadBalancer"]["ingress"]) == 1 and
        "elb.amazonaws.com" in status["loadBalancer"]["ingress"][0]["hostname"]
    )


@retry_unless(status_code=[403, 422], swallow_code=[409])
def create_pod_with_retry(kubectl, pod_spec):
    kubectl.api.create_namespaced_pod(pod_spec,
                                      namespace=TEST_NAMESPACE)


@retry_unless(status_code=[403, 422], swallow_code=[409, 404])
def delete_pod_with_retry(kubectl, pod_spec):
    kubectl.api.delete_namespaced_pod(body=V1DeleteOptions(),
                                      namespace=TEST_NAMESPACE,
                                      name=pod_spec.metadata.name)


@retry_unless(status_code=[403, 422], swallow_code=[409])
def create_pvc_with_retry(kubectl, pvc_spec):
    kubectl.api.create_namespaced_persistent_volume_claim(pvc_spec,
                                                          namespace=TEST_NAMESPACE)


@retry_unless(status_code=[403, 422], swallow_code=[409, 404])
def delete_pvc_with_retry(kubectl, pvc_spec):
    kubectl.api.delete_namespaced_persistent_volume_claim(body=V1DeleteOptions(),
                                                          name=pvc_spec.metadata.name,
                                                          namespace=TEST_NAMESPACE)


@retry_unless(status_code=[403, 422], swallow_code=[409])
def create_svc_with_retry(kubectl, svc_spec):
    from ax.kubernetes.client import KubernetesApiClient
    assert isinstance(kubectl, KubernetesApiClient)
    kubectl.api.create_namespaced_service(svc_spec,
                                          namespace=TEST_NAMESPACE)


@retry_unless(status_code=[403, 422], swallow_code=[409, 404])
def delete_svc_with_retry(kubectl, svc_spec):
    from ax.kubernetes.client import KubernetesApiClient
    assert isinstance(kubectl, KubernetesApiClient)
    kubectl.api.delete_namespaced_service(namespace=TEST_NAMESPACE,
                                          name=svc_spec.metadata.name)

kube_obj = {
    "name": "dummy",
    "kind": KubeApiObjKind.POD,
    "validator": wait_for_pod_validator
}


def yaml_to_swagger(kube_yaml_obj):
    kube_kind = kube_yaml_obj["kind"]
    (swagger_class_literal, swagger_instance) = KubeKindToV1KubeSwaggerObject[kube_kind]
    swagger_obj = ApiClient()._ApiClient__deserialize(kube_yaml_obj, swagger_class_literal)
    assert isinstance(swagger_obj, swagger_instance), \
        "{} has instance {}, expected {}".format(swagger_obj, type(swagger_obj), swagger_instance)
    return swagger_obj


def test_pod_create_successful(monitor, kubectl):
    pod = "test_pod_success"
    pod_file = PWD + "/testdata/" + pod + ".yml"

    pod_name = "{}-{:08d}".format(pod.replace("_", "-"), random.randint(1, 99999999))
    kube_obj["name"] = pod_name
    waiter = KubeObjWaiter()
    monitor.wait_for_kube_object(kube_obj=kube_obj, timeout=DEFAULT_POD_CREATION_TIMEOUT, waiter=waiter)

    with open(pod_file, "r") as f:
        data = f.read()
    yaml_obj = [obj for obj in yaml.load_all(data)]
    assert len(yaml_obj) == 1, "Loaded more than 1 yaml obj {}".format(yaml_obj)
    swagger_obj = yaml_to_swagger(yaml_obj[0])
    swagger_obj.metadata.name = pod_name

    create_pod_with_retry(kubectl, swagger_obj)

    waiter.wait()

    delete_pod_with_retry(kubectl, swagger_obj)

    if waiter.result != KubeObjStatusCode.OK:
        logger.info("Pod created with status %s, events: \n%s", waiter.result, pformat(waiter.details))
    test_result = waiter.result == KubeObjStatusCode.OK or waiter.result == KubeObjStatusCode.WARN
    assert test_result


def test_pod_create_invalid_img(kubectl, monitor):
    pod = "test_pod_invalid_img"
    pod_file = PWD + "/testdata/" + pod + ".yml"

    pod_name = "{}-{:08d}".format(pod.replace("_", "-"), random.randint(1, 99999999))
    kube_obj["name"] = pod_name
    waiter = KubeObjWaiter()
    monitor.wait_for_kube_object(kube_obj=kube_obj, timeout=60, waiter=waiter)

    with open(pod_file, "r") as f:
        data = f.read()
    yaml_obj = [obj for obj in yaml.load_all(data)]
    assert len(yaml_obj) == 1, "Loaded more than 1 yaml obj {}".format(yaml_obj)
    swagger_obj = yaml_to_swagger(yaml_obj[0])
    swagger_obj.metadata.name = pod_name

    create_pod_with_retry(kubectl, swagger_obj)

    waiter.wait()

    delete_pod_with_retry(kubectl, swagger_obj)

    if waiter.result != KubeObjStatusCode.ERR_PLAT_LOAD_IMAGE:
        logger.info("Pod created with status %s, events:\n%s", waiter.result, pformat(waiter.details))
    test_result = waiter.result == KubeObjStatusCode.ERR_PLAT_LOAD_IMAGE
    assert test_result


def test_pod_create_invalid_cmd(kubectl, monitor):
    pod = "test_pod_invalid_command"
    pod_file = PWD + "/testdata/" + pod + ".yml"

    pod_name = "{}-{:08d}".format(pod.replace("_", "-"), random.randint(1, 99999999))
    kube_obj["name"] = pod_name
    waiter = KubeObjWaiter()
    monitor.wait_for_kube_object(kube_obj=kube_obj, timeout=DEFAULT_POD_CREATION_TIMEOUT, waiter=waiter)

    with open(pod_file, "r") as f:
        data = f.read()
    yaml_obj = [obj for obj in yaml.load_all(data)]
    assert len(yaml_obj) == 1, "Loaded more than 1 yaml obj {}".format(yaml_obj)
    swagger_obj = yaml_to_swagger(yaml_obj[0])
    swagger_obj.metadata.name = pod_name

    create_pod_with_retry(kubectl, swagger_obj)

    waiter.wait()

    delete_pod_with_retry(kubectl, swagger_obj)

    if waiter.result != KubeObjStatusCode.ERR_FATAL:
        logger.info("Pod created with status %s, events:\n%s", waiter.result, pformat(waiter.details))
    test_result = waiter.result == KubeObjStatusCode.ERR_FATAL
    assert test_result


def test_pod_create_insufficient_resource(kubectl, monitor):
    pod = "test_pod_insufficient_resource"
    pod_file = PWD + "/testdata/" + pod + ".yml"

    pod_name = "{}-{:08d}".format(pod.replace("_", "-"), random.randint(1, 99999999))
    kube_obj["name"] = pod_name
    waiter = KubeObjWaiter()
    monitor.wait_for_kube_object(kube_obj=kube_obj, timeout=DEFAULT_POD_CREATION_TIMEOUT, waiter=waiter)

    with open(pod_file, "r") as f:
        data = f.read()
    yaml_obj = [obj for obj in yaml.load_all(data)]
    assert len(yaml_obj) == 1, "Loaded more than 1 yaml obj {}".format(yaml_obj)
    swagger_obj = yaml_to_swagger(yaml_obj[0])
    swagger_obj.metadata.name = pod_name

    create_pod_with_retry(kubectl, swagger_obj)

    waiter.wait()

    delete_pod_with_retry(kubectl, swagger_obj)

    if waiter.result != KubeObjStatusCode.ERR_INSUFFICIENT_RESOURCE:
        logger.info("Pod created with status %s, events:\n%s", waiter.result, pformat(waiter.details))
    test_result = waiter.result == KubeObjStatusCode.ERR_INSUFFICIENT_RESOURCE
    assert test_result


def test_pod_create_timeout(monitor):
    # just don't create a pod
    pod = "test_pod_timeout"

    pod_name = "{}-{:08d}".format(pod.replace("_", "-"), random.randint(1, 99999999))
    kube_obj["name"] = pod_name
    waiter = KubeObjWaiter()
    monitor.wait_for_kube_object(kube_obj=kube_obj, timeout=10, waiter=waiter)

    waiter.wait()

    if waiter.result != KubeObjStatusCode.ERR_PLAT_TASK_CREATE_TIMEOUT:
        logger.info("Pod created with status %s, events:\n%s", waiter.result, pformat(waiter.details))
    test_result = waiter.result == KubeObjStatusCode.ERR_PLAT_TASK_CREATE_TIMEOUT
    assert test_result


def test_volume_create_delete(kubectl, monitor, kubepoll):
    pvc = "test_pvc"
    pvc_label = "app=testpvc"
    pvc_file = PWD + "/testdata/" + pvc + ".yml"
    pvc_name = "{}-{:08d}".format(pvc.replace("_", "-"), random.randint(1, 99999999))
    kube_obj["name"] = pvc_name
    kube_obj["kind"] = KubeApiObjKind.PVC
    kube_obj["validator"] = wait_for_pvc_validator

    waiter = KubeObjWaiter()
    monitor.wait_for_kube_object(kube_obj=kube_obj, timeout=DEFAULT_PVC_CREATION_TIMEOUT, waiter=waiter)

    with open(pvc_file, "r") as f:
        data = f.read()
    yaml_obj = [obj for obj in yaml.load_all(data)]
    assert len(yaml_obj) == 1, "Loaded more than 1 yaml obj {}".format(yaml_obj)
    swagger_obj = yaml_to_swagger(yaml_obj[0])
    swagger_obj.metadata.name = pvc_name
    # Manually patch access mode as swagger client mistakenly interprets this as map
    swagger_obj.spec.access_modes = ["ReadWriteOnce"]

    create_pvc_with_retry(kubectl, swagger_obj)

    waiter.wait()

    try:
        if waiter.result != KubeObjStatusCode.OK:
            logger.info("PVC created with status %s, events:\n%s", waiter.result, pformat(waiter.details))
        test_result = waiter.result == KubeObjStatusCode.OK
        assert test_result

        pvcs = kubepoll.poll_kubernetes_sync(KubeKind.PVC, TEST_NAMESPACE, pvc_label)
        pvc = None
        for p in pvcs.items:
            if p.metadata.name == pvc_name:
                pvc = p
                break

        assert pvc

        kube_obj["name"] = pvc.spec.volume_name
        kube_obj["kind"] = KubeApiObjKind.PV
        kube_obj["validator"] = pv_release_validator

        waiter = KubeObjWaiter()
        monitor.wait_for_kube_object(kube_obj=kube_obj, timeout=DEFAULT_PV_DELETE_TIMEOUT, waiter=waiter)

        delete_pvc_with_retry(kubectl, swagger_obj)
        waiter.wait()

        if waiter.result != KubeObjStatusCode.OK:
            logger.info("PVC created with status %s, events:\n%s", waiter.result, pformat(waiter.details))
        test_result = waiter.result == KubeObjStatusCode.OK or waiter.result == KubeObjStatusCode.WARN
        assert test_result
    except Exception as e:
        delete_pvc_with_retry(kubectl, swagger_obj)
        raise e


@pytest.mark.skip(reason="Can cause resource limit problem when running this test in production")
def test_svc_lb_create(kubectl, monitor):
    svc = "test_svc_lb"
    svc_file = PWD + "/testdata/" + svc + ".yml"

    svc_name = "{}-{:08d}".format(svc.replace("_", "-"), random.randint(1, 99999999))
    kube_obj["name"] = svc_name
    kube_obj["kind"] = KubeApiObjKind.SERVICE
    kube_obj["validator"] = wait_for_svc_lb_validator

    waiter = KubeObjWaiter()
    monitor.wait_for_kube_object(kube_obj=kube_obj, timeout=DEFAULT_SVC_CREATION_TIMEOUT, waiter=waiter)

    with open(svc_file, "r") as f:
        data = f.read()
    yaml_obj = [obj for obj in yaml.load_all(data)]
    assert len(yaml_obj) == 1, "Loaded more than 1 yaml obj {}".format(yaml_obj)
    swagger_obj = yaml_to_swagger(yaml_obj[0])
    swagger_obj.metadata.name = svc_name

    create_svc_with_retry(kubectl, swagger_obj)
    waiter.wait()

    delete_svc_with_retry(kubectl, swagger_obj)

    if waiter.result != KubeObjStatusCode.OK:
        logger.info("Service created with status %s, events:\n%s", waiter.result, pformat(waiter.details))
    test_result = waiter.result == KubeObjStatusCode.OK
    assert test_result


def test_kube_poll_pods(kubectl, kubepoll):
    kubectl_result = kubectl.api.list_namespaced_pod(namespace=KUBE_NAMESPACE)
    kubepoll_result = kubepoll.poll_kubernetes_sync(KubeKind.POD, KUBE_NAMESPACE)
    assert str(kubectl_result.items) == str(kubepoll_result.items)


def test_kube_poll_svc(kubectl, kubepoll):
    kubectl_result = kubectl.api.list_namespaced_service(namespace=KUBE_NAMESPACE)
    kubepoll_result = kubepoll.poll_kubernetes_sync(KubeKind.SERVICE, KUBE_NAMESPACE)
    assert str(kubectl_result.items) == str(kubepoll_result.items)


def test_kube_poll_namespace(kubectl, kubepoll):
    kubectl_result = kubectl.api.list_namespace()
    kubepoll_result = kubepoll.poll_kubernetes_sync(KubeKind.NAMESPACE, None)
    assert str(kubectl_result.items) == str(kubepoll_result.items)


def test_kube_poll_secrets(kubectl, kubepoll):
    kubectl_result = kubectl.api.list_namespaced_secret(namespace=KUBE_NAMESPACE)
    kubepoll_result = kubepoll.poll_kubernetes_sync(KubeKind.SECRET, KUBE_NAMESPACE)
    assert str(kubectl_result.items) == str(kubepoll_result.items)
