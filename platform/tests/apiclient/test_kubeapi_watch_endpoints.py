# -*- coding: utf-8 -*-
#
# Copyright 2015-2016 Applatix, Inc. All rights reserved.
#

import pytest

from ax.exceptions import AXKubeApiException
from ax.kubernetes.ax_kube_dict import KUBE_NO_NAMESPACE_SET
from ax.kubernetes.client import KubernetesApiClient
from .testdata import TEST_KUBE_WATCH_API_ENDPOINTS_V1, TEST_KUBE_WATCH_API_ENDPOINTS_V1_AUTOSCALE,\
    TEST_KUBE_WATCH_API_ENDPOINTS_V1_BATCH, TEST_KUBE_WATCH_API_ENDPOINTS_V1_EXTENSION

# This is only a bogus client that `does not` work. This test is testing one specific function
# inside KubernetesApiClient
kn = KubernetesApiClient(use_proxy=True)


def test_v1_api_generation():
    endpoints_v1 = set()
    for item in KubernetesApiClient.item_v1:
        endpoints_v1.add(kn._generate_api_endpoint(item=item))
        if item in KUBE_NO_NAMESPACE_SET:
            endpoints_v1.add(kn._generate_api_endpoint(item=item, name="{name}"))
        else:
            endpoints_v1.add(kn._generate_api_endpoint(item=item, namespace="{namespace}"))
            endpoints_v1.add(kn._generate_api_endpoint(item=item, namespace="{namespace}", name="{name}"))
    assert endpoints_v1 == TEST_KUBE_WATCH_API_ENDPOINTS_V1


def test_v1_ext_api_generation():
    endpoints_v1_beta = set()
    for item in KubernetesApiClient.item_v1_beta:
        endpoints_v1_beta.add(kn._generate_api_endpoint(item=item))
        if item in KUBE_NO_NAMESPACE_SET:
            endpoints_v1_beta.add(kn._generate_api_endpoint(item=item, name="{name}"))
        else:
            endpoints_v1_beta.add(kn._generate_api_endpoint(item=item, namespace="{namespace}"))
            endpoints_v1_beta.add(kn._generate_api_endpoint(item=item, namespace="{namespace}", name="{name}"))
    assert endpoints_v1_beta == TEST_KUBE_WATCH_API_ENDPOINTS_V1_EXTENSION


def test_v1_autoscaling_api_generation():
    endpoints_v1_autoscaling = set()
    for item in KubernetesApiClient.item_v1_auto_scale:
        endpoints_v1_autoscaling.add(kn._generate_api_endpoint(item=item))
        if item in KUBE_NO_NAMESPACE_SET:
            endpoints_v1_autoscaling.add(kn._generate_api_endpoint(item=item, name="{name}"))
        else:
            endpoints_v1_autoscaling.add(kn._generate_api_endpoint(item=item, namespace="{namespace}"))
            endpoints_v1_autoscaling.add(kn._generate_api_endpoint(item=item, namespace="{namespace}", name="{name}"))
    assert endpoints_v1_autoscaling == TEST_KUBE_WATCH_API_ENDPOINTS_V1_AUTOSCALE


def test_v1_batch_api_generation():
    endpoints_v1_batch = set()
    for item in KubernetesApiClient.item_v1_batch:
        endpoints_v1_batch.add(kn._generate_api_endpoint(item=item))
        if item in KUBE_NO_NAMESPACE_SET:
            endpoints_v1_batch.add(kn._generate_api_endpoint(item=item, name="{name}"))
        else:
            endpoints_v1_batch.add(kn._generate_api_endpoint(item=item, namespace="{namespace}"))
            endpoints_v1_batch.add(kn._generate_api_endpoint(item=item, namespace="{namespace}", name="{name}"))
    assert endpoints_v1_batch == TEST_KUBE_WATCH_API_ENDPOINTS_V1_BATCH


def test_invalid_item_api_generation():
    with pytest.raises(AXKubeApiException):
        kn.watch(item="invalid_item")


def test_invalid_namespace_api_generation():
    with pytest.raises(AXKubeApiException):
        kn.watch(item="persistentvolumes", namespace="default", name="test_pv")
    with pytest.raises(AXKubeApiException):
        kn.watch(item="namespaces", namespace="default", name="test_ns")
    with pytest.raises(AXKubeApiException):
        kn.watch(item="nodes", namespace="default", name="test_nodes")
    with pytest.raises(AXKubeApiException):
        kn.watch(item="thirdpartyresources", namespace="default", name="n")
