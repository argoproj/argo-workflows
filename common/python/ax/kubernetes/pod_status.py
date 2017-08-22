#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

"""
Module to get and parse status from Kubernetes pod.
"""
import json
import logging

from ax.kubernetes.swagger_client import V1Pod, V1PodStatus, V1ObjectMeta
from ax.util import const
from ax.util.const import KB, MiB

logger = logging.getLogger(__name__)


class PodStatus(object):
    def __init__(self, pod_status):
        # TODO: support get pod status later.
        assert isinstance(pod_status, V1Pod), "Invalid pod_status. {}; {}".format(type(pod_status), pod_status)
        self._pod_status = pod_status

    def list_current_containers(self, id_only=False):
        """
        Return a list of (cname, cid) for current containers
        :return:
        """
        status_info = self._pod_status.status
        assert isinstance(status_info, V1PodStatus)
        current_containers = []

        # Get a list of running pods and a list of terminated pods
        for c in status_info.container_statuses:
            cur_cid = None
            cur_cname = None
            try:
                cur_cid = c.container_id[len("docker://"):]
                cur_cname = c.name
            except AttributeError:
                pass
            if cur_cid and cur_cname:
                current_containers.append(cur_cid if id_only else (cur_cname, cur_cid))
        return current_containers

    def get_app_meta(self):
        """
        Parse metadata as if the pod is an application pod
        :return: cost_id, app_id, dep_id, dep_name
        """
        meta = self._pod_status.metadata
        assert isinstance(meta, V1ObjectMeta), "Expect pod meta to be V1ObjectMeta, but get {}.".format(type(meta))
        labels = meta.labels
        assert isinstance(labels, dict), "Expect pod labels to be dict, but get {}.".format(type(labels))
        dep_name = str(labels["deployment"])

        annotations = meta.annotations
        assert isinstance(annotations, dict), "Expect pod annotation to be dict, but get {}.".format(type(annotations))

        cost_id = str(annotations["ax_costid"])
        ax_identifiers = json.loads(annotations["AX_IDENTIFIERS"])
        app_id = str(ax_identifiers["application_id"])
        dep_id = str(ax_identifiers["deployment_id"])

        return cost_id, app_id, dep_id, dep_name

    def get_resources_for_container(self, containername):
        """
        Get pod resource request based on pod status spec.
        :param containername: Name of container
        :return tuple for cpu and memory. CPU is number of cores. Memory is in MiB.
        """
        for c in self._pod_status.spec.containers:
            if c.name == containername:
                res = c.resources.requests
                if res is None:
                    return 0.0, 0.0
                else:
                    cpu = res.get("cpu", "0m")
                    if cpu[-1:] != "m":
                        cpu = "{}m".format(cpu)
                    cpu = int(cpu[:-1]) / float(KB)
                    membytes = self._kubernetes_mem_to_int(res.get("memory", "0.0Mi"))
                    mem_mib = membytes / MiB
                    return cpu, mem_mib
        else:
            logger.debug("Container %s not in pod %s any more.", containername, self._pod_status.metadata.name)
            return 0.0, 0.0

    def get_container_name_for_id(self, containerid):
        id_str = "docker://{}".format(containerid)
        for c in self._pod_status.status.container_statuses or []:
            if c.container_id and c.container_id == id_str:
                return c.name

        return None

    # Limits and requests for memory are measured in bytes.
    # Memory can be expressed a plain integer or as fixed-point integers
    # with one of these SI suffixes (E, P, T, G, M, K) or their power-of-two
    # equivalents (Ei, Pi, Ti, Gi, Mi, Ki). For example, the following represent
    # roughly the same value: 128974848, 129e6, 129M , 123Mi.
    @staticmethod
    def _kubernetes_mem_to_int(s):
        try:
            # try to see if it is plain number
            return int(float(s))
        except ValueError:
            if s[-1] == "i":
                base = s[:-2]
                multiplier = getattr(const, s[-2:] + "B")
            else:
                base = s[:-1]
                multiplier = getattr(const, s[-1:] + "B")
            return int(float(base) * multiplier)
