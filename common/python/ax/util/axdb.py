#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2016 Applatix, Inc. All rights reserved.
#

"""
Client library to access axdb.

Supports kv table type only. Value must be dict.

Sample usage:

from ax.util.axdb import AXDBClient

db = AXDBClient("axdb", "axsys/config")
key = "my key"
val = {"data": "data"}
db.kv_set(key, val)
"""

import logging
import requests
import json
import time
import uuid

from six import string_types
from ax.exceptions import AXTimeoutException

logger = logging.getLogger(__name__)

AXDB_HOST = "axdb.axsys"
AXDB_PORT = 8083
AXDB_VERSION = "1"


class AXDBClient(object):
    """
    Client to access axdb.
    """
    default_max_retry = 10

    def __init__(self, host, table,
                 port=AXDB_PORT,
                 version=AXDB_VERSION,
                 cluster_id=None):
        """
        Initializing AXDB URL.

        :param host: Hostname for AXDB
        :param table: Table name in axdb server. Must be key value table type.
        :param port: Port number, default settings number, 8080
        :param version: Version number, default to settings number, 0.1
        :return: None
        """
        self._host = host
        self._table = table
        self._port = port
        self._version = version
        self._cluster_id = cluster_id
        self._db = ""

    def _regen_db_url(self):
        if self._host is None:
            host = AXDB_HOST
        else:
            host = self._host

        self._db = "http://" + str(host) + ":" + str(self._port) + "/" + "v" + str(self._version) + "/" + self._table
        return True

    @staticmethod
    def get_local_axdb_client(table, version=AXDB_VERSION):
        return AXDBClient(host=AXDB_HOST, table=table, port=AXDB_PORT, version=version)

    def get_url(self):
        if not self._db:
            self._regen_db_url()
        return self._db

    @staticmethod
    def _retry_sleep(max_retry, retry_count):
        # xxx todo, use max_retry and retry_count to determine how long to sleep
        time.sleep(2)

    @staticmethod
    def _check_request_uuid(response, request_uuid):
        request_uuid_echo = response.headers.get("X-Request-UUID-Echo", None)
        if request_uuid_echo:
            assert request_uuid == request_uuid_echo, "X-Request-UUID not matching, {} vs {}".format(request_uuid, request_uuid_echo)
        else:
            logger.debug("no request_uuid_echo, %s", response.headers)

    def _get(self, url_part2, max_retry, is_kv=True):
        retry_count = 0
        while retry_count <= max_retry:
            if not self._db or retry_count != 0:
                self._regen_db_url()
            retry_count += 1
            url = self._db + url_part2
            try:
                request_uuid = str(uuid.uuid1())
                headers = {"X-Request-UUID": request_uuid}
                r = requests.get(url, headers=headers)
                self._check_request_uuid(r, request_uuid)
                if r.status_code != requests.codes.ok:
                    logger.warning("Status code %s for get %s", r.status_code, url)
                    return None
                else:
                    try:
                        if is_kv:
                            j = r.json()
                            if isinstance(j, list) and len(j) >= 1:
                                assert isinstance(j[0], dict), "{} is not dict for {}".format(j[0], url)
                                v = j[0].get("value", None)
                                assert isinstance(v, string_types), "{} is invalid for {}".format(j[0], url)
                                return json.loads(v)
                            else:
                                logger.debug("empty result %s for %s", j, url)
                                return None
                        else:
                            val = r.json()
                            return val
                    except ValueError:
                        logger.exception("Failed to parse response %s, url=%s, retry=%s", r.json(), url, retry_count)
                        # keep retry
                        self._retry_sleep(max_retry=max_retry, retry_count=retry_count)
                        continue
            except requests.RequestException:
                logger.exception("Failed to get from %s, retry=%s", url, retry_count)
                # keep retry
                self._retry_sleep(max_retry=max_retry, retry_count=retry_count)
                continue
            except Exception:
                # raise other exception
                logger.exception("Other exception, failed to get from %s", url)
                raise

        logger.error("max retry reached to get %s", url)
        raise AXTimeoutException("axdb get")

    def _set(self, data, url_part2, is_post, max_retry):
        retry_count = 0
        while retry_count <= max_retry:
            if not self._db or retry_count != 0:
                self._regen_db_url()
            retry_count += 1
            url = self._db + url_part2
            try:
                request_uuid = str(uuid.uuid1())
                headers = {"X-Request-UUID": request_uuid}
                if is_post:
                    r = requests.post(self._db, json=data, headers=headers)
                else:
                    r = requests.put(self._db, json=data, headers=headers)
                self._check_request_uuid(r, request_uuid)
                if r.status_code == requests.codes.forbidden:
                    assert is_post, "not post: invalid status code {} for set {}".format(r.status_code, url)
                    logger.warning("post: status code %s for set %s", r.status_code, url)
                    return False
                elif r.status_code == requests.codes.ok:
                    return True
                else:
                    logger.warning("status code %s for set %s. %s", r.status_code, url, is_post)
                    return False
            except requests.RequestException:
                logger.exception("Failed to set %s, retry=%s", url, retry_count)
                # keep retry
                self._retry_sleep(max_retry=max_retry, retry_count=retry_count)
                continue
            except Exception:
                logger.exception("Other exception, failed to set %s", url)
                raise

        logger.error("max retry reached to set %s", url)
        raise AXTimeoutException("axdb set")

    def kv_get(self, key, max_retry=default_max_retry):
        """
        Get value from axdb kv table.

        :param key: Key (string) from kv table.
        :param max_retry: number of max retry
        :return: value as json object, or None on failure.
        """
        # AXDB URL is in key=xxx format.
        url_part2 = "?key=" + key
        return self._get(url_part2=url_part2, is_kv=True, max_retry=max_retry)

    def kv_set(self, key, val, allow_over_write=True, max_retry=default_max_retry):
        """
        Set key value in kv table.

        :param key: Key (string) in kv table.
        :param val: Value as dict.
        :param allow_over_write: whether to allow overwrite.
        :param max_retry: number of max retry
        :return: True if successful, or False if not.
        """
        # Dump first for inner json and then wrap it around with outer key/value json.
        # logger.debug("Setting AXDB kv key=%s val=%s", key, json.dumps(val))
        data = {"key": key, "value": json.dumps(val)}
        return self._set(data=data, url_part2="?key=" + key, is_post=(not allow_over_write), max_retry=max_retry)

    def kv_delete(self, key, max_retry=default_max_retry):
        """
        Set key value in kv table.

        :param key: Key (string) in kv table.
        :param max_retry: number of max retry
        :return: True if successful, or False if not.
        """
        # Dump first for inner json and then wrap it around with outer key/value json.
        logger.debug("Delete AXDB kv key=%s", key)
        data = [{"key": key}]
        retry_count = 0

        while retry_count <= max_retry:
            if not self._db or retry_count != 0:
                self._regen_db_url()
            retry_count += 1
            url = self._db + "?key=" + key
            request_uuid = str(uuid.uuid1())
            headers = {"X-Request-UUID": request_uuid}
            try:
                r = requests.delete(self._db, json=data, headers=headers)
                self._check_request_uuid(r, request_uuid)
                if r.status_code == requests.codes.ok:
                    logger.debug("Delete %s successful.", url)
                    return True
                else:
                    logger.warning("status code %s for delete %s.", r.status_code, url)
                    return False
            except requests.RequestException:
                logger.exception("Failed to delete %s, retry=%s", url, retry_count)
                # keep retry
                self._retry_sleep(max_retry=max_retry, retry_count=retry_count)
                continue
            except Exception:
                logger.exception("Failed to delete %s", url)
                raise

        logger.error("max retry reached to delete %s", url)
        raise AXTimeoutException("axdb delete")

    def insert(self, data, max_retry=default_max_retry):
        """
        Insert one item in table. Use POST.
        :param data: Dict for data.
        :param max_retry: number of max retry
        :return: True if successful, or False if not.
        """
        return self._set(data=data, url_part2="", is_post=True, max_retry=max_retry)

    def get(self, data, max_retry=default_max_retry):
        """
        Get item matching key specified by data
        :param data: Dict for keys
        :param max_retry: number of max retry
        :return: List of items. return None on error
        """
        url_part2 = "?"
        for d in data:
            url_part2 += "%s=%s" % (d, data[d])
            url_part2 += "&"

        return self._get(url_part2=url_part2, is_kv=False, max_retry=max_retry)
