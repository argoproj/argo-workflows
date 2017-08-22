#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

"""
Main program of applet
"""

import logging
import time
from threading import Thread

from .appdb import ApplicationRecord
from .amclient import ApplicationManagerClient
from .protocol import DeploymentNannyProtocol
from .plm_pool import PodLogManagerPool
from .handshake import AXHandshakeServer
from .consts import *
from ax.kubernetes.kubelet import KubeletClient
from ax.kubernetes.client import KubernetesApiClient, retry_unless
from ax.kubernetes.pod_status import PodStatus
from ax.kubernetes.swagger_client import V1Pod, V1ObjectMeta


logger = logging.getLogger(__name__)
logger.setLevel(logging.DEBUG)


class Applet(object):
    def __init__(self):
        self._kubectl = KubernetesApiClient()

        # Initialize kubelet client singleton
        self._kubelet = KubeletClient()

        # Initialize DB, PLM pool,
        self._app_record = ApplicationRecord(table_create=True)
        self._plm_pool = PodLogManagerPool()

        # Initialize handshake server
        self._handshake_server = AXHandshakeServer(sock_addr=APPLET_SOCK,
                                                   proto=DeploymentNannyProtocol)

        # Initialize application monitor client
        self._am = ApplicationManagerClient()

        # In case applet restarts, it should continue to nanny existing pods
        self._nanny_existing_pods()

    def _nanny_existing_pods(self):
        """
        If there are records in DB already, start PodLogManager for these pods
        :return:
        """
        logger.info("Nannying existing pods ...")
        db_records = self._app_record.load_from_db()
        for k in db_records.keys():
            # TODO: store AID, WID, RID and stuff in db and get these values from db
            app_name = db_records[k]["app"]
            app_id = db_records[k]["aid"]
            pod_name = db_records[k]["pod"]
            dep_name = db_records[k]["dep"]
            dep_id = db_records[k]["did"]
            plm_to_add = []
            for c in db_records[k]["containers"]:
                plm_to_add.append((c["name"], c["id"]))
            self._plm_pool.create_or_update_pod_log_manager(
                app_name=app_name,
                app_id=app_id,
                deployment_name=dep_name,
                deployment_id=dep_id,
                pod_name=pod_name,
                to_add=plm_to_add
            )

    def run(self):
        """
        Start main logic of applet. As the handshake server is handling
        signals, it has to be in side main thread
        :return:
        """
        try:
            SyncPods(self._kubelet).start()
            self._handshake_server.start_server()
        except Exception as e:
            logger.exception("Applet failed to start due to \"%s\"!!! R.I.P", e)


class SyncPods(Thread):
    def __init__(self, kubelet):
        super(SyncPods, self).__init__()
        self._app_record = None
        self._plm_pool = PodLogManagerPool()
        self._amcli = ApplicationManagerClient()
        self._kubelet = kubelet

    def _load_pods_from_kube(self):
        kube_records = {}
        for p in self._kubelet.list_namespaced_pods(label_selectors=["application=*"]):
            assert isinstance(p, V1Pod)
            assert isinstance(p.metadata, V1ObjectMeta)
            key = "{}.{}".format(p.metadata.name, p.metadata.namespace)
            kube_records[key] = p
        return kube_records

    def _sync_one_pod(self, app_name, app_id, dep_name, dep_id, pod_name, db_to_add, plm_to_add, to_delete):
        self._plm_pool.create_or_update_pod_log_manager(
            app_name=app_name,
            app_id=app_id,
            deployment_name=dep_name,
            deployment_id=dep_id,
            pod_name=pod_name,
            to_add=plm_to_add,
            to_remove=to_delete
        )
        self._app_record.refresh_db_record(
            to_add=db_to_add,
            to_delete=to_delete
        )

    @staticmethod
    def _generate_pod_diff(pod_in_db, pod_in_kube, app_name, app_id, dep_name, dep_id, pod_name):
        """
        For "Current" containers, we start new log collectors
        For "Stale" containers, we stop log collectors
        :param pod_in_db: dict such as
            {
                "app": AppName,
                "pod": PodName,
                "aid": ApplicationId,
                "did": DeploymentId,
                "dep": DeploymentName,
                "containers": [
                    {
                        "name": ContainerName,
                        "id": ContainerId,
                        "last": LastRotatedLog
                    },
                    ...
                ]
            }
        :return: db_to_add, plm_to_add, to_delete
        """
        ps = PodStatus(pod_in_kube)
        cur_cid = ps.list_current_containers(id_only=True)
        plm_to_add = PodStatus(pod_in_kube).list_current_containers()

        cid_to_cname = {}
        for cname, cid in plm_to_add:
            cid_to_cname[cid] = cname

        cid_in_db = [c["id"] for c in pod_in_db["containers"]]

        db_to_add = [(CUR_RECORD_VERSION, app_name, app_id, dep_name, dep_id, pod_name, cid_to_cname[cid], cid, "")
                     for cid in list(set(cur_cid) - set(cid_in_db))]
        to_delete = [(CUR_RECORD_VERSION, app_name, pod_name, c) for c in list(set(cid_in_db) - set(cur_cid))]
        return db_to_add, plm_to_add, to_delete

    def _do_sync(self):
        logger.info("Syncing pods with kubelet ...")

        db_records = self._app_record.load_from_db()
        if not db_records:
            return

        # TODO: parallelize this with a proper working Queue
        # TODO: for pods known to kubelet but not known to DB, send heartbeat if pod is not healthy
        timestamp = int(time.time())

        # Sync pods both known to kubernetes and to applet
        for pod in self._kubelet.list_namespaced_pods(label_selectors=["application=*"]):
            assert isinstance(pod, V1Pod)
            assert isinstance(pod.metadata, V1ObjectMeta)
            key = "{}.{}".format(pod.metadata.name, pod.metadata.namespace)
            if key in db_records:
                app_name = db_records[key]["app"]
                app_id = db_records[key]["aid"]
                pod_name = db_records[key]["pod"]
                dep_name = db_records[key]["dep"]
                dep_id = db_records[key]["did"]

                db_to_add, plm_to_add, to_delete = self._generate_pod_diff(
                    pod_in_db=db_records[key],
                    pod_in_kube=pod,
                    app_name=app_name,
                    app_id=app_id,
                    dep_name=dep_name,
                    dep_id=dep_id,
                    pod_name=pod_name
                )

                self._sync_one_pod(
                    app_name=app_name,
                    app_id=app_id,
                    dep_name=dep_name,
                    dep_id=dep_id,
                    pod_name=pod_name,
                    db_to_add=db_to_add,
                    plm_to_add=plm_to_add,
                    to_delete=to_delete
                )

                try:
                    self._amcli.send_heart_beat(
                        app_name=app_name,
                        pod_name=pod_name,
                        dep_id=dep_id,
                        hb_type=HeartBeatType.HEART_BEAT,
                        timestamp=timestamp,
                        pod=pod
                    )
                except Exception as e:
                    logger.error("Cannot send heartbeat. Error: %s", e)

                db_records.pop(key, None)

        # Delete records known only to applet
        for key in db_records:
            app_name = db_records[key]["app"]
            pod_name = db_records[key]["pod"]
            dep_id = db_records[key]["did"]
            to_delete = [(CUR_RECORD_VERSION, app_name, pod_name, c["id"]) for c in db_records[key]["containers"]]
            self._plm_pool.remove_pod_log_manager(app_name, pod_name)
            self._app_record.refresh_db_record(
                to_add=[],
                to_delete=to_delete
            )
            try:
                self._amcli.send_heart_beat(
                    app_name=app_name,
                    pod_name=pod_name,
                    dep_id=dep_id,
                    hb_type=HeartBeatType.TOMB_STONE,
                    timestamp=timestamp,
                    pod=None
                )
            except Exception as e:
                logger.error("Cannot send heartbeat. Error: %s", e)

    def run(self):
        logger.info("\n\n======= Pod Sync loop started =======\n")
        self._app_record = ApplicationRecord()
        while True:
            try:
                self._do_sync()
            except Exception as e:
                logger.exception("Error in applet pod sync loop: %s", e)
            time.sleep(APPLET_SYNC_PERIOD)







