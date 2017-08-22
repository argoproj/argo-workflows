#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2016 Applatix, Inc. All rights reserved.
#

import json
import logging
import requests

from retrying import retry

from ax.util import RetryWrapper

logger = logging.getLogger(__name__)


class AxdbConstants(object):
    '''
    Definateion of the constants for AXDB component
    '''
    RestStatusOK        = requests.codes.ok           # for all successes
    RestStatusInvalid   = requests.codes.bad_request  # for invalid parameters (GET) or JSON body
    RestStatusDenied    = requests.codes.unauthorized # for unauthorized access
    RestStatusForbidden = requests.codes.forbidden    # request is not forbidden. When POSTing to KeyValue store but the key exists already
    RestStatusNotFound  = requests.codes.not_found    # for invalid url

    TableTypeTimeSeries  = 0     # time series, use POST to generate a new object with ax_uuid returned. Use PUT to update existing objects
    TableTypeKeyValue    = 1     # key value pair, use POST to create new object, PUT to update
    TableTypeTimedKeyValue = 2   # key value with ax_time managed by AXDB. ax_time is required for query (primary key).
    TableTypeCounter     = 3     # counter, new insert will increment a counter, which will be returned

    ColumnTypeString    = 0
    ColumnTypeDouble    = 1
    ColumnTypeInteger   = 2
    ColumnTypeBoolean   = 3
    ColumnTypeArray     = 4
    ColumnTypeMap       = 5
    # ColumnTypeTimestamp = 6
    ColumnTypeUUID      = 7

    ColumnIndexNone       = 0  # not a key
    ColumnIndexStrong     = 1  # we will do query on this column, and we expect high cardinality
    ColumnIndexWeak       = 2  # we will do query on this column, and we expect low cardinality
    ColumnIndexClustering = 3  # Use this for clustering index, can specify multiple columns
    ColumnIndexPartition  = 4  # Use this as partition key, can specify multiple columns

    TaskStatusWaiting     = 'WAITING'    # task is waiting to run
    TaskStatusRunning     = 'RUNNING'    # task is running
    TaskStatusRetry       = 'RETRY'      # task is failed, but retry automatically
    TaskStatusPrelim      = 'PRELIM'     # task initiated by a flow manager, place holder entry on the AXDB
    TaskStatusComplete    = 'COMPLETE'   # task complete

    TaskResultSuccess     = 'SUCCESS'    # task completed: successful
    TaskResultFailure     = 'FAILURE'    # task completed: failed
    TaskResultCancelled   = 'CANCELLED'  # task completed: cancelled


class AXDB(object):

    def __init__(self, url):
        '''
        :param url:
        :return:
        '''
        if len(url) == 0:
            self._url = 'http://localhost:8080/v1'
        else:
            self._url = url

        self.requests = RetryWrapper(requests, decorator=retry(wait_exponential_multiplier=100, stop_max_attempt_number=10))

    def post(self, table_name, data_map):
        '''
        :param table_name:
        :param data_map:
        :return:
        '''
        resp = self.requests.post(self._url + table_name, json.dumps(data_map))
        if resp.status_code != AxdbConstants.RestStatusOK:
            logger.info('post to table {}, got status code {} '.format(table_name, resp.status_code))
        return resp

    def put(self, table_name, data_map):
        '''
        :param table_name:
        :param data_map:
        :return:
        '''
        resp = self.requests.put(self._url + table_name, json.dumps(data_map))
        if resp.status_code != AxdbConstants.RestStatusOK:
            logger.info('post to table {}, got status code {} '.format(table_name, resp.status_code))
        return resp

    def get(self, payload):
        '''
        :param payload:
        :return:
        '''
        resp = self.requests.get(self._url + payload)
        if resp.status_code != AxdbConstants.RestStatusOK:
            logger.info('AXDB request failure with status code {}'.format(resp.status_code))
        return resp

    def delete(self, table_name):
        '''
        :param table_name:
        :return:
        '''
        resp = self.requests.delete(self._url + table_name)
        if resp.status_code != AxdbConstants.RestStatusOK:
            logger.info('AXDB request failure with status code {}'.format(resp.status_code))
        return resp

    def version(self):
        '''
        :return:
        '''
        return self.get('/axdb/version')

    def create_table(self, table_map):
        '''
        :param table_map:
        :return:
        '''
        return self.put("/axdb/update_table", table_map)

    def does_table_exist(self, table_name):
        '''
        :param table_name:
        :return:

        Note: The normal table_name format is /<APPName>/<Name>, e.g. /axops/commit,
              here it will automatically convert to the internal format:  <APPName>_<Name>
              for example: "/axops/commit" to "axops_commit"
        '''
        _table_name = table_name.strip('/').replace('/', '_')
        payload = '/axint/table_definition?key={}'.format(_table_name)
        return bool(self.get(payload))
