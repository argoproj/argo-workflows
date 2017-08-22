#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

"""
To create / update / delete a pool of PodLogManagers
"""

import logging
from threading import Lock
from future.utils import with_metaclass

from ax.util.singleton import Singleton
from ax.platform.sidecar import PodLogManager
from ax.platform.container_specs import is_ax_aux_container


logger = logging.getLogger(__name__)
logger.setLevel(logging.DEBUG)


class PodLogManagerPool(with_metaclass(Singleton)):
    def __init__(self):
        self._lock = Lock()
        self._plms = {}

    @staticmethod
    def _do_plm_update(plm, to_add, to_remove):
        logger.info("Adding log collectors for container %s", to_add)
        for cname, cid in to_add or []:
            if not is_ax_aux_container(cname):
                try:
                    plm.start_log_watcher(cname, cid)
                except Exception as e:
                    logger.warning("Cannot start log watcher for container %s (%s). Error: %s", cname, cid, e)

        logger.info("Removing log collectors for container %s", to_remove)
        for cid in to_remove or []:
            try:
                plm.stop_log_watcher(cid)
            except Exception as e:
                logger.exception("Cannot stop log watcher for container %s. Error: %s", cid, e)

    def get_containers(self):
        """
        Get a list of containers whose logs we are monitoring.
        This is mostly used for tests
        :return:
        """
        rst = []
        with self._lock:
            for key in self._plms:
                plm = self._plms[key]
                assert isinstance(plm, PodLogManager)
                rst.extend(plm.get_containers())
        return rst

    def get_plm_number(self):
        return len(self._plms)

    def create_or_update_pod_log_manager(self, app_name, app_id, deployment_name, deployment_id,
                                         pod_name, to_add=None, to_remove=None):
        """
        Start PodLogManager for given app/pod, with information provided from
        pod_meta, start collecting logs for container ids in to_add, stop collecting logs
        for container ids in to_remove
        :param app_name:
        :param app_id:
        :param deployment_name:
        :param deployment_id:
        :param pod_name:
        :param to_add: list of (cname, cid) tuple
        :param to_remove: list of cid
        :return:
        """
        plm_key = "{pname}.{app}".format(pname=pod_name, app=app_name)
        logger.info("Create or update PLM %s with app_name(%s), app_id(%s), dep_name(%s), dep_id(%s).\n\nPLM adding %s;\nPLM removing %s\n",
                    plm_key, app_name, app_id, deployment_name, deployment_id, to_add, to_remove)

        with self._lock:
            plm = self._plms.get(plm_key, None)
            if not plm:
                # PLM is not there, this is a newly registered pod, so start log management
                logger.info("Creating new plm %s", plm_key)
                self._plms[plm_key] = PodLogManager(pod_name=pod_name,
                                                    service_id=deployment_id,
                                                    root_id=app_id,
                                                    leaf_full_path=deployment_name,
                                                    namespace=app_name,
                                                    app_mode=True)
            else:
                logger.info("PLM %s already exists, proceed to update", plm_key)
            self._do_plm_update(self._plms[plm_key], to_add, to_remove)

    def remove_pod_log_manager(self, app_name, pod_name):
        plm_key = "{pname}.{app}".format(pname=pod_name, app=app_name)
        # No lock is needed  here as this function is only called when
        # pod is gone. Same pod won't be created again, so the caller
        # is the only one who operate on this particular PLM object
        plm = self._plms.get(plm_key, None)
        if plm:
            assert isinstance(plm, PodLogManager)
            logger.info("Deleting Pod Log Manager %s.%s from record", pod_name, app_name)
            # Terminate is sync, it waits for all collectors to join
            plm.terminate()
            del self._plms[plm_key]
        else:
            logger.info("Pod Log Manager %s.%s has already been deleted", pod_name, app_name)
