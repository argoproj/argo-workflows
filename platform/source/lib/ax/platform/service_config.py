#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2016 Applatix, Inc. All rights reserved.
#

import logging
import ast
import re

from ax.platform.ax_monitor_helper import KubeObjStatusCode
from ax.platform.pod import SIDEKICK_WAIT_CONTAINER_NAME

from ax.kubernetes import swagger_client
from ax.kubernetes.client import KubernetesApiClient
from ax.kubernetes.pod_status import PodStatus

logger = logging.getLogger(__name__)


class AXKubeServiceConfig(object):
    def __init__(self):
        self._kubectl = KubernetesApiClient(use_proxy=True)
        self._container_info = {}

    def register_container(self, cid, container):
        """
        container object:
        {
            "name": "",
            "kube_pod_name": "",
            "kube_namespace": ""
        }
        :param cid: container id
        :param container: container info
        :return:
        """
        assert cid, "Registering container without an id"
        assert container, "Registering container without container info"
        cost_id, service_id, cpu, memory = self._get_container_devops_info(cid, container)
        info = {
            "cost_id": cost_id,
            "service_id": service_id,
            "cpu_request": cpu,
            "memory_request": memory
        }
        info.update(container)
        # Update local info first and then assign to avoid partial update if get devops info fails.
        self._container_info[cid] = info
        logger.info("Registered container %s %s", info["name"], cid)

    def unregister_container(self, cid):
        try:
            del self._container_info[cid]
        except KeyError as ke:
            logger.exception("Failed to deregister container %s, error: %s", cid, ke)
        logger.info("Unregistered container %s", cid)

    def get_cost_id(self, cid):
        return self._container_info[cid]["cost_id"]

    def get_service_id(self, cid):
        return self._container_info[cid]["service_id"]

    def get_cpu_request(self, cid):
        return self._container_info[cid]["cpu_request"]

    def get_memory_request(self, cid):
        return self._container_info[cid]["memory_request"]

    def _get_container_devops_info(self, container_id, container):
        import json
        pod = self._kubectl.api.read_namespaced_pod_status(namespace=container["kube_namespace"],
                                                           name=container["kube_pod_name"])
        assert isinstance(pod, swagger_client.V1Pod), "Got invalid Pod object form Kubernetes api server"
        if SIDEKICK_WAIT_CONTAINER_NAME in container["name"] or "artifacts" in container["name"]:
            # Special casing these user-space containers
            cost_id = {
                "user": "axsys",
                "app": "system",
                "service": "artifacts-management"
            }
        else:
            cost_id = self._get_cost_id_from_pod(pod, container["kube_namespace"])
        service_id = self._get_service_id_from_pod(pod, container["kube_namespace"])
        # Need short container name to match resource spec.

        pstatus = PodStatus(pod)
        containername = pstatus.get_container_name_for_id(container_id)
        cpu, memory = pstatus.get_resources_for_container(containername)
        return cost_id, service_id, cpu, memory

    @staticmethod
    def _get_cost_id_from_pod(pod, namespace="axsys"):
        if namespace == "axsys":
            return {
                "user": "axsys",
                "app": "system",
                "service": pod.metadata.labels.get("app", None)
            }
        elif namespace == "kube-system":
            return {
                "user": "k8s",
                "app": "system",
                "service": pod.metadata.labels.get("k8s-app", "k8s-sys")
            }
        else:
            cost_id = pod.metadata.annotations.get("ax_costid")
            if cost_id is None or cost_id == "None":
                # cost_id can be set to string "None" in some test cases.
                return {
                    "user": KubeObjStatusCode.UNKNOWN,
                    "app": KubeObjStatusCode.UNKNOWN,
                    "service": KubeObjStatusCode.UNKNOWN
                }
            else:
                return ast.literal_eval(cost_id)

    @staticmethod
    def _get_service_id_from_pod(pod, namespace="axsys"):
        if namespace == "axsys" or namespace == "kube-system":
            return None
        return pod.metadata.annotations.get("ax_serviceid", None)

