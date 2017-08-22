#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

"""
Interface to local database to manipulate persisted records
"""

import logging
import sqlite3
from retrying import retry
from pprint import pformat

from .consts import *

logger = logging.getLogger(__name__)
logger.setLevel(logging.DEBUG)


class ApplicationRecord(object):
    def __init__(self, db=APPLET_DB, table_create=False):
        self._conn = sqlite3.connect(db)
        if table_create:
            self._db_init()

    def _db_init(self):
        c = self._conn.cursor()
        c.execute(
            '''
            CREATE TABLE IF NOT EXISTS app_records (
                version text,
                app_name text,
                app_id text,
                deployment_name text,
                deployment_id text,
                pod_name text,
                container_name text,
                container_id text,
                last_done text,
                PRIMARY KEY (version, app_name, pod_name, container_id)
            )
            '''
        )
        self._commit_with_retry()

    def close_connection(self):
        self._conn.close()

    @retry(
        wait_fixed=1000,
        stop_max_attempt_number=3
    )
    def _delete_db_records(self, to_delete):
        """
        Remove db entries
        :param to_delete: list of (version, app_name, pod_name, container_id) tuples
        :return:
        """
        if not to_delete:
            return
        try:
            c = self._conn.cursor()
            c.executemany(
                '''
                DELETE FROM app_records WHERE
                version=? AND
                app_name=? AND
                pod_name=? AND
                container_id=?
                ''',
                to_delete
            )
        except Exception as e:
            logger.exception("Failed to execute delete with values %s. Error: %s", to_delete, e)
            self._conn.rollback()
            raise e

    @retry(
        wait_fixed=1000,
        stop_max_attempt_number=3
    )
    def _add_db_records(self, to_add):
        """
        Add db entries
        :param to_add: (version, app_name, pod_name, container_id, last_done) tuples
        :return:
        """
        if not to_add:
            return
        try:
            c = self._conn.cursor()
            c.executemany("INSERT OR IGNORE INTO app_records VALUES (?,?,?,?,?,?,?,?,?)", to_add)
        except Exception as e:
            logger.exception("Failed to execute insersion with values %s. Error: %s", to_add, e)
            self._conn.rollback()
            raise e

    @retry(
        wait_fixed=1000,
        stop_max_attempt_number=3
    )
    def _commit_with_retry(self):
        try:
            self._conn.commit()
        except Exception as e:
            logger.exception("DB commit failure with error %s. Rolling back and retrying ...", e)
            self._conn.rollback()
            raise e

    def load_from_db(self):
        """
        Load pod / log upload info from db into self._cache
        This assumes db has been initialized
        :return:
        {
            "PodName.AppName": {
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
            ...
        }
        """
        record = {}
        c = self._conn.cursor()
        for (app, aid, dep, did, pod, cname, cid, last) in c.execute(
            '''
            SELECT app_name, app_id, deployment_name, deployment_id, pod_name, container_name, container_id, last_done
            FROM app_records
            WHERE
            version=?
            ''',
            (CUR_RECORD_VERSION,)
        ):
            key = "{}.{}".format(pod, app)
            if not record.get(key, None):
                record[key] = {
                    "app": app,
                    "pod": pod,
                    "aid": aid,
                    "did": did,
                    "dep": dep,
                    "containers": [{
                        "name": cname,
                        "id": cid,
                        "last": last
                    }]
                }
            else:
                record.get(key).get("containers").append({
                    "name": cname,
                    "id": cid,
                    "last": last
                })
        logger.debug("Current records in DB (%s in total):\n%s", len(record), pformat(record))
        return record

    def update_application(self, app_name, app_id, deployment_name, deployment_id, pod_name, cur_containers):
        """
        Update a single application:
        Add DB entry for cur_containers; remove DB entry for last_containers
        :return:
        """
        to_add = [(CUR_RECORD_VERSION, app_name, app_id, deployment_name, deployment_id, pod_name, cname, cid, "")
                  for cname, cid in cur_containers]
        self.refresh_db_record(to_add, [])

    def refresh_db_record(self, to_add=None, to_delete=None):
        """
        Remove entry in DB for our dated pods
        :param to_add: list of primary key tuple (version, app_name, pod_name, container_id) to be added
        :param to_delete: list of primary key tuple (version, app_name, pod_name, container_id) to be deleted
        :return:
        """
        if not to_add and not to_delete:
            logger.info("Nothing to refresh")
            return

        self._add_db_records(to_add)
        self._delete_db_records(to_delete)

        self._commit_with_retry()
        logger.info("Successfully refreshed DB.\n\nAdded DB records:\n%s\nDeleted records:\n%s\n",
                    pformat(to_add), pformat(to_delete))

    @retry(
        wait_fixed=1000,
        stop_max_attempt_number=3
    )
    def record_done_log(self, app_name, pod_name, container_id, log_full_path):
        """
        When one ContainerLogCollector thread finishes to process a log (uploaded to s3 / reported
        to artifact-manager), we persist a record in DB
        :param app_name:
        :param pod_name:
        :param container_id:
        :param log_full_path:
        :return:
        """
        c = self._conn.cursor()
        c.execute(
            '''
            UPDATE app_records SET last_done=?
            WHERE
            version=? AND
            app_name=? AND
            pod_name=? AND
            container_id=?
            ''',
            (log_full_path, CUR_RECORD_VERSION, app_name, pod_name, container_id)
        )
        self._commit_with_retry()
        logger.info("Successfully added log(%s) to app(%s), pod(%s), container(%s)",
                    log_full_path, app_name, pod_name, container_id)

    def get_last_done(self, app_name, pod_name, container_id):
        """
        Check if md5sum is in db. Cache lookup is perfectly fine here
        :param app_name:
        :param pod_name:
        :param container_id:
        :return:
        """
        c = self._conn.cursor()
        c.execute(
            '''
            SELECT last_done FROM app_records
            WHERE
            version=? AND
            app_name=? AND
            pod_name=? AND
            container_id=?
            ''',
            (CUR_RECORD_VERSION, app_name, pod_name, container_id)
        )
        rst = c.fetchall()
        if rst:
            (lastdone, ) = rst[0]
            return lastdone
        return None


