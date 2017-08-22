#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2016 Applatix, Inc. All rights reserved.
#

"""
This module serializes kubernetes `list` api calls, which would
reduces Kubernetes API server load
"""
import logging
from threading import Semaphore

from ax.kubernetes.ax_kube_dict import KubeKind, AXNameSpaces
from ax.util.callbacks import Callback
from ax.util.singleton import Singleton
from future.utils import with_metaclass
from retrying import retry


logger = logging.getLogger(__name__)


class KubeObjPollResultCarrier(object):
    def __init__(self):
        self._result = None

    @property
    def result(self):
        return self._result

    @result.setter
    def result(self, rst):
        self._result = rst


class KubeObjPoll(with_metaclass(Singleton, Callback)):

    def __init__(self, kubectl):
        super(KubeObjPoll, self).__init__()
        self.kubectl = kubectl
        self.add_cb(self._poll_kube_obj_cb)
        self.start()

    def poll_kubernetes_sync(self, kind, namespace=AXNameSpaces.AXSYS, label_selector=""):
        sem = Semaphore(0)
        result_carrier = KubeObjPollResultCarrier()
        self.post_event(self.kubectl, kind, result_carrier, namespace, sem, label_selector)
        sem.acquire()
        return result_carrier.result

    @staticmethod
    def _poll_kube_obj_cb(kubectl, kind, result_carrier, namespace, sem=None, label_selector=""):

        @retry(wait_exponential_multiplier=1000, stop_max_attempt_number=3)
        def _call_kube_api(kubectl, kind, result_carrier, namespace, label_selector):
            endpoints = {
                KubeKind.SERVICE: kubectl.api.list_namespaced_service,
                KubeKind.POD: kubectl.api.list_namespaced_pod,
                KubeKind.PVC: kubectl.api.list_namespaced_persistent_volume_claim,
                KubeKind.SECRET: kubectl.api.list_namespaced_secret,
                KubeKind.NAMESPACE: kubectl.api.list_namespace,
                KubeKind.NODE: kubectl.api.list_node,
                KubeKind.CONFIGMAP: kubectl.api.list_namespaced_config_map
            }

            if namespace:
                result_carrier.result = endpoints[kind](namespace, label_selector=label_selector)
            else:
                result_carrier.result = endpoints[kind](label_selector=label_selector)

        try:
            _call_kube_api(kubectl, kind, result_carrier, namespace, label_selector)
        except Exception as e:
            logger.error("Unable to call kube api with error %s", e)
        if sem:
            sem.release()
