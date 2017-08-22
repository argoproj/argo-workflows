#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2016 Applatix, Inc. All rights reserved.
#

"""
AX platform events/object/task monitor service
"""

import copy
import logging
import uuid

from threading import RLock, Timer
from future.utils import with_metaclass

from ax.kubernetes.ax_kube_dict import KubeKind, KubeApiObjKind, AXNameSpaces
from ax.kubernetes.client import KubernetesApiClient
from ax.platform.exceptions import AXPlatformException
from ax.util.singleton import Singleton
from .ax_monitor_helper import KubeObjMonitor, KubeObjStatusCode, KubeObjStatus

logger = logging.getLogger(__name__)


class KubeEventsMonitor(KubeObjMonitor):
    def __init__(self, name, kubectl, namespace=AXNameSpaces.AXUSER):
        super(KubeEventsMonitor, self).__init__(name, KubeApiObjKind.EVENT, kubectl, namespace)
        self.record = AXKubeWaitingRecord()
        self.supported_objects = frozenset([KubeKind.POD, KubeKind.PVC, KubeKind.PV, KubeKind.SERVICE])

    def _do_monitor(self, kubeobj):
        event = kubeobj.get("object", None)
        if not event or "involvedObject" not in event:
            return

        kind = event["involvedObject"].get("kind", None)
        name = event["involvedObject"].get("name", None)
        reason = event.get("reason", None)
        message = event.get("message", None)
        simplified_event = {"name": name, "reason": reason, "message": message}
        category = KubeObjStatusCode.categorize_event(kind, reason, message)
        if category == KubeObjStatusCode.OK:
            return

        if kind in self.supported_objects:
            if category == KubeObjStatusCode.WARN or category == KubeObjStatusCode.ERR_OTHER:
                if not self.record.post_event_to_kubeobj(name, simplified_event):
                    logger.debug("Thread %s: caught %s event for Kubeobj kind [%s]. Event: %s", self.name, category,
                                 kind, str(simplified_event))

                if reason == "ErrInvalidSignature":
                    # Wrong volume mounted, even it is posted to object, lets print it out
                    logger.warn("Thread %s: caught %s event for Kubeobj kind [%s]. Event: %s", self.name, category,
                                kind, str(simplified_event))
            elif category == KubeObjStatusCode.AX_INTERNAL:
                logger.debug("Thread %s: caught %s event for Kubeobj kind [%s]. Event: %s", self.name, category,
                            kind, str(simplified_event))
            else:
                # ERR_INSUFFICIENT_RESOURCE and ERR_FATAL
                wait_objs = self.record.get_kubeobj_status(name)
                for wait_obj in wait_objs:
                    wait_obj_ret = self.record.remove_kubeobj_status(wait_obj.name, wait_obj.uid)
                    if wait_obj_ret:
                        self._do_process_critical_event(wait_obj_ret, kind, simplified_event, category)
                    else:
                        logger.error("Thread %s: caught %s event for Kubeobj kind [%s]. Event: %s", self.name, category,
                                     kind, str(simplified_event))
        else:
            if category == KubeObjStatusCode.ERR_FATAL:
                logger.error("Thread %s: caught %s event for Kubeobj kind [%s]. Event: %s", self.name, category, kind,
                             str(simplified_event))

    def _do_process_critical_event(self, obj, kind, event, ev_category):
        logger.warning("Thread %s: Kubeobj kind [%s]: Critical event %s happened. Evoking callback for %s %s. Event: %s",
                       self.name, kind, ev_category, kind, obj.name, str(event))
        obj.reason.append(event)
        obj.waiter.callback(result=ev_category, detail=obj.reason)


class KubePodsMonitor(KubeObjMonitor):
    def __init__(self, name, kubectl, namespace=AXNameSpaces.AXUSER):
        super(KubePodsMonitor, self).__init__(name, KubeApiObjKind.POD, kubectl, namespace)
        self.record = AXKubeWaitingRecord()

    def _do_monitor(self, kubeobj):
        name = kubeobj["object"]["metadata"].get("name", None)
        jobname = None
        if "labels" in kubeobj["object"]["metadata"]:
            jobname = kubeobj["object"]["metadata"]["labels"].get("job-name", None)
        if not name:
            return
        if jobname:
            name = jobname
        status = kubeobj["object"].get("status", None)
        if not status:
            return
        wait_objs = self._check_kubeobj_status(name, status)
        if wait_objs:
            self._process_finished_pods(wait_objs)
        self._check_container_oom(status)

    def _process_finished_pods(self, objs):
        for obj in objs:
            logger.info("Thread %s: object %s:%s has reached expected status", self.name, obj.name, obj.kind)
            if len(obj.reason) == 0:
                obj.waiter.callback(result=KubeObjStatusCode.OK, detail=obj.reason)
            else:
                obj.waiter.callback(result=KubeObjStatusCode.WARN, detail=obj.reason)

    def _check_container_oom(self, pod_status):
        containers = pod_status.get("containerStatuses", None)
        if containers:
            for c in containers:
                state = c.get("state", None)
                if state:
                    if "terminated" in state.keys() and state["terminated"].get("reason", None) == "OOMKilled":
                        logger.warning("Container %s has been OOM Killed. Detail: %s", c["name"], str(c))

    def _check_kubeobj_status(self, name, status):
        result = []
        wait_objs = self.record.get_kubeobj_status(name)
        for wait_obj in wait_objs:
            if wait_obj.kind == self.kind:
                try:
                    if wait_obj.validator(status):
                        r = self.record.remove_kubeobj_status(wait_obj.name, wait_obj.uid)
                        if r:
                            result.append(r)
                except Exception as e:
                    logger.exception("User validator has the following exception {} for {} with status {}".format(e, name, status))
        return result


class KubeObjDefaultMonitor(KubeObjMonitor):
    def __init__(self, name, kind, kubectl, namespace=None):
        super(KubeObjDefaultMonitor, self).__init__(name, kind, kubectl, namespace)
        self.record = AXKubeWaitingRecord()

    def _do_monitor(self, kubeobj):
        name = kubeobj["object"]["metadata"].get("name", None)
        if not name:
            return
        status = kubeobj["object"].get("status", None)
        if not status:
            return
        wait_objs = self._check_kubeobj_status(name, status)
        for wait_obj in wait_objs:
            self._process_finished_obj(wait_obj)

    def _check_kubeobj_status(self, name, status):
        result = []
        wait_objs = self.record.get_kubeobj_status(name)
        for wait_obj in wait_objs:
            if wait_obj.kind == self.kind:
                try:
                    if wait_obj.validator(status):
                        r = self.record.remove_kubeobj_status(wait_obj.name, wait_obj.uid)
                        if r:
                            result.append(r)
                except Exception as e:
                    logger.exception(
                        "User validator has the following exception {} for {} with status {}".format(e, name, status))
        return result

    def _process_finished_obj(self, wait_obj):
        logger.info("Thread %s: object %s:%s has reached expected status", self.name, wait_obj.name, wait_obj.kind)
        if len(wait_obj.reason) == 0:
            wait_obj.waiter.callback(result=KubeObjStatusCode.OK, detail=wait_obj.reason)
        else:
            wait_obj.waiter.callback(result=KubeObjStatusCode.WARN, detail=wait_obj.reason)


class AXKubeWaitingRecord(with_metaclass(Singleton, object)):

    def __init__(self):
        self._wait_list = {}
        self._wait_list_lock = RLock()


    def post_kubeobj_status(self, status):
        """
        A simple lock free post - It's safe
        :param status: KubeObjStatus
        :return:
        """

        with self._wait_list_lock:
            if status.name in self._wait_list:
                self._wait_list[status.name].append(status)
            else:
                self._wait_list[status.name] = [status]

    def get_kubeobj_status(self, name):
        """
        Note this is an UGLY workaround. As Kubernetes assigns name and uid
        to object (i.e. pod of a job) at run time, we can only know what the
        name would contain by the time we register the waiter. As long as
        the number of object waiting is not big, we won't sacrifice much
        performance. We still use a map here because we might need to support
        waiting for object using tag, which would be an exact match.
        Same to remove_kubeobj_status() below.

        :param name: name of the kubeobj status
        :return:
        """
        ret = []
        with self._wait_list_lock:
            for k, value in self._wait_list.items():
                if k in name: # partial match
                    ret = ret + copy.copy(value)

        return ret

    def remove_kubeobj_status(self, name, uid):
        """
        Used for removing waiting object
        :param name: name of the object
        :param uid: uuid of the object
        :return: KubeObjStatus
        """
        with self._wait_list_lock:
            if name in self._wait_list:
                objs = self._wait_list[name]
                if not objs:
                    logger.warning("uuid %s return empty %s", name, objs)
                for obj in objs:
                    if uid == obj.uid:
                        # match
                        if obj.timer is not None:
                            obj.timer.cancel()
                        objs.remove(obj)
                        if len(objs) == 0:
                            self._wait_list.pop(name)
                        return obj
                    else:
                        logger.debug("uuid %s not match %s for %s", uid, obj.uid, name)
            else:
                logger.debug("cannot find %s in %s", name, self._wait_list.keys())

        return None

    def post_event_to_kubeobj(self, name, event):
        """
        Post an event to a waiting kubeobj

        :param name:
        :param event:
        :return: True if there is an waiting obj with name `name` and event is posted, False otherwise
        """
        wait_objs = self.get_kubeobj_status(name)
        for wait_obj in wait_objs:
            wait_obj.reason.append(event)

        return len(wait_objs) > 0

    def empty(self):
        with self._wait_list_lock:
            if len(self._wait_list) > 0:
                return False
        return True

    def clear_all(self):
        with self._wait_list_lock:
            for k, objs in self._wait_list:
                for obj in objs:
                    if obj.timer:
                        obj.timer.cancel()

        self._wait_list.clear()


class AXKubeMonitor(with_metaclass(Singleton, object)):

    def __init__(self, kubectl=None, config_file=None):
        if kubectl:
            self.kubectl = kubectl
        elif config_file:
            self.kubectl = KubernetesApiClient(config_file=config_file)
        else:
            self.kubectl = KubernetesApiClient(host="localhost", port="8001", use_proxy=True)

        self.monitors = []
        self.record = AXKubeWaitingRecord()
        self.started = False

    def reload_monitors(self, namespace=AXNameSpaces.AXUSER):
        self.monitors = []
        self.monitors.append(KubeEventsMonitor(self._gen_monitor_name(KubeApiObjKind.EVENT), self.kubectl, namespace=namespace))
        self.monitors.append(KubePodsMonitor(self._gen_monitor_name(KubeApiObjKind.POD), self.kubectl, namespace=namespace))
        self.monitors.extend(
            [
                KubeObjDefaultMonitor(self._gen_monitor_name(KubeApiObjKind.PVC), KubeApiObjKind.PVC, self.kubectl, namespace=namespace),
                KubeObjDefaultMonitor(self._gen_monitor_name(KubeApiObjKind.PV), KubeApiObjKind.PV, self.kubectl, namespace=namespace),
                KubeObjDefaultMonitor(self._gen_monitor_name(KubeApiObjKind.SERVICE), KubeApiObjKind.SERVICE, self.kubectl, namespace=namespace)
            ]
        )

    @staticmethod
    def _gen_monitor_name(kubekind):
        return "kube-" + kubekind + "-monitor"

    def _validate_kube_object(self, kube_obj):
        if kube_obj:
            if kube_obj["kind"] and kube_obj["name"] and kube_obj["validator"]:
                if self.kubectl.validate_object(kube_obj["kind"]):
                    return True
        return False

    @staticmethod
    def _is_err_img_pull(msg):
        """
        Determines if it is failed image pull based on last event message
        """
        return True if "Back-off pulling image" in msg \
                       or "Failed to pull image" in msg \
                       or "ErrImagePull" in msg \
            else False

    def process_timeout(self, name, uuid):
        wait_obj = self.record.remove_kubeobj_status(name, uuid)
        if wait_obj:
            logger.warning("Kubeobj %s:%s timeout", wait_obj.name, wait_obj.kind)
            if len(wait_obj.reason) > 0:
                if self._is_err_img_pull(wait_obj.reason[-1]["message"]):
                    wait_obj.waiter.callback(result=KubeObjStatusCode.ERR_PLAT_LOAD_IMAGE, detail=wait_obj.reason)
                    return

            wait_obj.waiter.callback(result=KubeObjStatusCode.ERR_PLAT_TASK_CREATE_TIMEOUT, detail=wait_obj.reason)

    def wait_for_kube_object(self, kube_obj=None, timeout=None, waiter=None):
        """
        kube_obj example:
        {
            "kind": "pods",
            "name": "my-pod",
            "validator": <some_lambda_function>
        }

        :param kube_obj:
        :param timeout: timeout value in seconds
        :param waiter: AXWaiter object
        :return:
        """
        if not self._validate_kube_object(kube_obj):
            msg = "Invalid kube_object: {}".format(str(kube_obj))
            raise AXPlatformException(msg)

        if not waiter:
            raise AXPlatformException("No waiter specified for wait_for_kube_object")

        uid = str(uuid.uuid4())
        if timeout:
            timer = Timer(timeout, self.process_timeout, (kube_obj["name"], uid))
        else:
            timer = None
        status = KubeObjStatus(name=kube_obj["name"],
                               kind=kube_obj["kind"],
                               validator=kube_obj["validator"],
                               waiter=waiter,
                               uid=uid,
                               timer=timer)
        if timer:
            timer.start()

        self.record.post_kubeobj_status(status)

    def start(self):
        if not self.started:
            for m in self.monitors:
                m.start()
            self.started = True

    def stop(self, force=False):
        if not self.started:
            return
        if not force and not self.record.empty():
            raise AXPlatformException("Waiter pending, need to force stop")
        self.record.clear_all()
        for m in self.monitors:
            m.request_stop()
        self.started = False
