#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2016 Applatix, Inc. All rights reserved.
#

"""
AX platform events/object/task monitor service helper classes
"""

import abc
import logging
import time

from future.utils import with_metaclass
from threading import Thread, Semaphore

from ax.kubernetes.ax_kube_dict import KubeKind, KUBE_NO_NAMESPACE_SET

logger = logging.getLogger(__name__)


class ABCWaiterBase(with_metaclass(abc.ABCMeta, object)):

    @abc.abstractmethod
    def callback(self, result, detail):
        pass


class KubeObjWaiter(ABCWaiterBase):
    def __init__(self):
        self._sem = Semaphore(0)
        self._result = KubeObjStatusCode.UNKNOWN
        self._detail = []

    def callback(self, result, detail):
        self._result = result
        self._detail = detail
        self._sem.release()
        logger.debug("Default AX Waiter callback gets called. Result = {}; Detail = {}".format(result, detail))

    def wait(self):
        self._sem.acquire()

    @property
    def result(self):
        """Get object's result string"""
        return self._result

    @property
    def details(self):
        """Get object's result detail"""
        return self._detail


class KubeObjMonitor(with_metaclass(abc.ABCMeta, Thread)):

    def __init__(self, name, kind, kubectl, namespace=None):
        super(KubeObjMonitor, self).__init__()
        self._name = name
        self._kind = kind
        self._kubectl = kubectl
        self.daemon = True
        self._request_stop = False
        if kind in KUBE_NO_NAMESPACE_SET:
            self._namespace = None
        else:
            self._namespace = namespace

    def run(self):
        logger.info("Thread %s: Starts", self._name)
        while True:
            try:
                objs = self._kubectl.watch(item=self._kind, namespace=self._namespace)
                for obj in objs:
                    if self._request_stop:
                        logger.info("Thread %s quitting upon request", self._name)
                        return
                    self._do_monitor(obj)
            except Exception as e:
                if self._request_stop:
                    logger.info("Thread %s quitting upon request", self._name)
                    return
                logger.exception("Thread %s: got exception %s", self.name, str(e))
                time.sleep(10)

    def request_stop(self):
        self._request_stop = True

    @property
    def name(self):
        """Get thread name"""
        return self._name

    @property
    def kind(self):
        """Get thread kind"""
        return self._kind

    @abc.abstractmethod
    def _do_monitor(self, kubeobj):
        """
        Kubernetes object monitor logic

        :param kubeobj: Kubernetes object, i.e. a pod
        :return:
        """
        pass


class KubeObjStatusCode(object):
    OK = "OK"
    WARN = "WARN"
    OBJ_EXISTS = "OBJ_EXISTS"
    ERR_FATAL = "ERR_FATAL"
    ERR_PLAT_TASK_CREATE_TIMEOUT = "ERR_PLAT_TASK_CREATE_TIMEOUT"
    ERR_INSUFFICIENT_RESOURCE = "ERR_INSUFFICIENT_RESOURCE"
    ERR_PLAT_LOAD_IMAGE = "ERR_PLAT_LOAD_IMAGE"
    ERR_OTHER = "ERR_OTHER"
    DELETED = "DELETED"
    AX_INTERNAL = "AX_INTERNAL"
    UNKNOWN = "UNKNOWN"
    HEALTHY = "HEALTHY"
    UNHEALTHY = "UNHEALTHY"

    @classmethod
    def categorize_event(cls, kind, reason, message):
        """

        :param kind: KubeWatchEvent["object"]["involvedObject"]["kind"]
        :param reason: KubeWatchEvent["object"]["reason"]
        :param message: KubeWatchEvent["object"]["message"]
        :return: error literal
        """
        if kind == KubeKind.POD:
            return KubePodEventCategorizer.categorize_pod_event(reason, message)
        elif kind == KubeKind.PVC or kind == KubeKind.PV:
            return KubePVCEventCategorizer.categorize_pvc_event(reason)
        elif kind == KubeKind.SERVICE:
            return KubeSVCEventCategorizer.categorize_svc_event(reason)
        else:
            return KubeGeneralEventCategorizer.categorize_general_event(reason)


class KubePodEventCategorizer(object):
    warn_list = frozenset(["Killing", "BackOff", "FailedScheduling", "FailedSync", "InfraChanged", "FailedMount", "ErrInvalidSignature"])
    fatal_list = frozenset(["Failed", "InspectFailed", "ErrImageNeverPull", "FailedCreate", "FailedDelete"])
    white_list = frozenset(["Created", "Started", "Pulling", "Pulled", "Scheduled", "TriggeredScaleUp"])
    ax_list = frozenset(["CreateSignature", "FoundSignature"])

    @classmethod
    def categorize_pod_event(cls, reason, message):
        if reason in cls.warn_list:
            if reason == "FailedScheduling" and "Insufficient" in message:
                return KubeObjStatusCode.ERR_INSUFFICIENT_RESOURCE
            return KubeObjStatusCode.WARN
        elif reason in cls.fatal_list:
            # An image might finally get pulled even if it might fail several times
            if reason == "Failed" and "Failed to pull image" in message:
                return KubeObjStatusCode.WARN
            else:
                return KubeObjStatusCode.ERR_FATAL
        elif reason in cls.white_list:
            return KubeObjStatusCode.OK
        elif reason in cls.ax_list:
            return KubeObjStatusCode.AX_INTERNAL
        else:
            return KubeObjStatusCode.ERR_OTHER


class KubePVCEventCategorizer(object):
    fatal_list = frozenset(["ClaimLost", "ProvisioningFailed"])
    warn_list = frozenset(["VolumeFailedDelete"])

    @classmethod
    def categorize_pvc_event(cls, reason):
        if reason in cls.fatal_list:
            return KubeObjStatusCode.ERR_FATAL
        elif reason in cls.warn_list:
            return KubeObjStatusCode.WARN
        else:
            return KubeObjStatusCode.OK


class KubeGeneralEventCategorizer(object):

    warn_list = frozenset(["NodeNotReady", "NodeNotSchedulable", "NodeRebooted", "HostPortConflict",
                           "NodeSelectorMismatching", "NilShaper"])
    fatal_list = frozenset(["HostNetworkNotSupported", "KubeletSetupFailed", "InvalidDiskCapacity",
                            "FreeDiskSpaceFailed"])
    insufficient_resource_list = frozenset(["InsufficientFreeCPU", "InsufficientFreeMemory", "OutOfDisk"])

    misc_list = frozenset(["Unhealthy", "FailedValidation", "FailedPostStartHook", "FailedPreStopHook",
                           "LoadBalancerUpdateFailed", "MissingClusterDNS"])

    @classmethod
    def categorize_general_event(cls, reason):
        if reason in cls.warn_list:
            return KubeObjStatusCode.WARN
        elif reason in cls.fatal_list:
            return KubeObjStatusCode.ERR_FATAL
        elif reason in cls.insufficient_resource_list:
            return KubeObjStatusCode.ERR_INSUFFICIENT_RESOURCE
        elif reason in cls.misc_list:
            return KubeObjStatusCode.ERR_OTHER
        else:
            return KubeObjStatusCode.OK


class KubeSVCEventCategorizer(object):
    white_list = frozenset(["CreatingLoadBalancer", "CreatedLoadBalancer", "DeletingLoadBalancer",
                            "DeletedLoadBalancer"])

    @classmethod
    def categorize_svc_event(cls, reason):
        if reason in cls.white_list:
            return KubeObjStatusCode.OK
        else:
            return KubeObjStatusCode.WARN


class KubeObjStatus(object):
    def __init__(self, name, kind, validator, waiter, uid, timer=None):
        self.name = name
        self.kind = kind
        self.validator = validator

        self.code = KubeObjStatusCode.ERR_FATAL
        # List of kube events whose status are in nonfatal_task_events
        # and fatal_task_events
        self.reason = []
        self.waiter = waiter
        self.uid = uid
        self.timer = timer

