#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

"""
Client for talking with local kubelet
"""

import logging
import requests
import ijson
from future.utils import with_metaclass

from ax.cloud import Cloud
from ax.util.singleton import Singleton
from ax.kubernetes.swagger_client import ApiClient
from ax.kubernetes.swagger_client import V1Pod

logger = logging.getLogger(__name__)


class KubeletClient(with_metaclass(Singleton)):
    # Read only port is documented in http://kubernetes.io/docs/admin/kubelet/
    KUBELET_RO_PORT = 10255

    def __init__(self, host_ip=None):

        self._host_ip = host_ip if host_ip else Cloud().meta_data().get_private_ip()
        assert self._host_ip, "Kubelet Client is not properly initialized: Missing host ip"
        logger.info("Kubelet client uses host ip %s", self._host_ip)
        self._kubelet_url = "http://{}:{}".format(self._host_ip, self.KUBELET_RO_PORT)

    def list_pods(self):
        re = requests.get(self._kubelet_url + "/pods", stream=True)
        re.raise_for_status()
        for pod in ijson.items(re.raw, "items.item") or []:
            swagger_pod = ApiClient()._ApiClient__deserialize(pod, "V1Pod")
            yield swagger_pod
        return

    def list_namespaced_pods(self, namespace=None, name=None, label_selectors=None):
        """
        Return pods match all not-None arguments
        :param namespace:
        :param name:
        :param label_selectors: a list of key=value pairs
        :return:
        """
        logger.info("Listing pods with criteria: NameSpace(%s); Name(%s); LabelSelector(%s)",
                     namespace, name, label_selectors)
        for pod in self.list_pods():
            assert isinstance(pod, V1Pod)
            pod_labels = pod.metadata.labels or {}
            label_match = True
            for label in label_selectors or []:
                k, v = label.split("=")
                kube_label_val = pod_labels.get(k, None)
                # TODO: change it to RegEx
                if not((v == "*" and kube_label_val) or (kube_label_val == v)):
                    label_match = False
                    break
            if not label_match:
                continue
            if namespace and pod.metadata.namespace != namespace:
                continue
            if name and pod.metadata.name != name:
                continue
            logger.debug("Pod matches, yielding...")
            yield pod
        return
