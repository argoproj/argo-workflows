#!/usr/bin/env python
#
# Copyright 2015-2016 Applatix, Inc.  All rights reserved.
#
# -*- coding: utf-8 -*-
#
import copy
import json
import logging
import time
from retrying import retry
from requests.exceptions import HTTPError

from ax.devops.axdb.constants import AxdbConstants
from ax.devops.axrequests.axrequests import AxRequests
from ax.exceptions import AXException, AXIllegalArgumentException
from ax.devops.settings import AxSettings
from ax.devops.utility.utilities import retry_on_errors

logger = logging.getLogger(__name__)


class BaseAxdbClient(object):
    """Base client to AXDB."""

    def __init__(self, host=AxSettings.AXDB_HOSTNAME, port=AxSettings.AXDB_PORT, version=AxSettings.AXDB_VERSION, url=None):
        """Initialize the AXDB client.

        :param host:
        :param port:
        :param version:
        :param url:
        :return:
        """
        self.ax_request = AxRequests(host, port=port, version=version, protocol='http', url=url)

    @property
    def version(self):
        """Get AXDB version.

        :return:
        """
        version = self.ax_request.get('/axdb/version', value_only=True)  # Example: [{u'version': u'v0.1'}]
        return version[0].get('version', None)

    def create_table(self, schema):
        """Create a table in AXDB. This function will call update_table, which will create the table if not existed.

        :param schema: Table schema which is a dictionary.
        :return:
        """
        return self.update_table(schema)

    def update_table(self, schema):
        """Update a table in AXDB.

        :param schema: Table schema which is a dictionary.
        :return:
        """
        return self.ax_request.put('/axdb/update_table', data=json.dumps(schema))

    def delete_table(self, table_name):
        """Delete a table in AXDB.

        :param table_name: Name of table.
        :return:
        """
        return self.ax_request.delete(table_name)

    def has_table(self, table_name):
        """Test whether a table with given table name exists in AXDB.

        :param table_name:
        :return: Boolean.
        """
        # Need to convert table name /<AppName>/<Name> to ax_key '<AppName>_<Name>' first
        ax_key = table_name.strip('/').replace('/', '_')
        return bool(self.ax_request.get('/axint/table_definition?ax_key={}'.format(ax_key), value_only=True))

    def create_entry(self, table_name, payload, **kwargs):
        """Create an entry in an AXDB table.

        :param table_name:
        :param payload:
        :return:
        """
        return self.ax_request.post(table_name, data=json.dumps(payload), **kwargs)

    def update_entry(self, table_name, payload, **kwargs):
        """Update an entry in an AXDB table.

        :param table_name:
        :param payload:
        :return:
        """
        return self.ax_request.put(table_name, data=json.dumps(payload), **kwargs)

    def delete_entry(self, table_name, payload, **kwargs):
        """Delete an entry in an AXDB table.

        :param table_name:
        :param payload:
        :return:
        """
        return self.ax_request.delete(table_name, data=json.dumps(payload), **kwargs)

    def has_entry(self, table_name, payload, **kwargs):
        """Test whether entries satisfying the condition specified in the payload exist in the table.

        :param table_name:
        :param payload:
        :return: True / False.
        """
        return bool(self.query(table_name, payload, **kwargs).json())

    def init_entry(self, table_name, payload=None, **kwargs):
        """Initialize entry.

        :param table_name:
        :param payload:
        :return: The axdb UUID, or None if fails.
        """
        if payload is None:
            payload = {}
        resp = self.ax_request.post(table_name, data=json.dumps(payload), **kwargs)
        return resp.json().get('ax_uuid')

    def query(self, table_name, payload=None, **kwargs):
        """Execute a query.

        :param table_name:
        :param payload:
        :param kwargs:
        :return:
        """
        return self.ax_request.get(table_name, params=payload, **kwargs)

    def retry_request(self, method, table_name, params=None, data=None, **kwargs):
        """
        A retry wrapper for AXDB client.

        For different methods, the retry scenario is different. Thus, the underline AxRequest class is not
        suitable for differentiating these scenarios. On the other hand, the response object has all the
        information we need. By specifying a retry_on_exception, the client method can specify when to perform
        retry based on the exception from the remote API.

        Currently, the retry mechanism for AXDB client is controlled by the following parameters:
        - Exponential backoff: Wait 2^x * 1000 milliseconds between each retry, up to 1 min, then 1 min afterwards
        - Max tries: 20 (roughly retry for at most 15 minutes)
        - Retry based on result: parameter retry_on_exception.

        :param method:
        :param table_name:
        :param params:
        :param data:
        :param kwargs:
        :return:
        """

        wait_exponential_multiplier = kwargs.pop('wait_exponential_multiplier', 1000)
        wait_exponential_max = kwargs.pop('wait_exponential_max', 60000)
        stop_max_attempt_number = kwargs.pop('max_retry', 20)
        retry_on_exception = kwargs.pop('retry_on_exception', None)

        @retry(wait_exponential_multiplier=wait_exponential_multiplier, wait_exponential_max=wait_exponential_max,
               stop_max_attempt_number=stop_max_attempt_number, retry_on_exception=retry_on_exception)
        def _run():
            # TODO: maybe remove the usage of ax_request later
            return self.ax_request._run_requests(method, table_name, params=params, data=data, **kwargs)

        if data:
            data = json.dumps(data)
        return _run()

    @staticmethod
    def retry_function(f, *args, **kwargs):
        """Retry a function call

        :param f:
        :param args:
        :param kwargs:
        :returns:
        """
        wait_exponential_multiplier = kwargs.pop('wait_exponential_multiplier', 1000)
        wait_exponential_max = kwargs.pop('wait_exponential_max', 60000)
        stop_max_attempt_number = kwargs.pop('max_retry', 20)
        retry_on_exception = kwargs.pop('retry_on_exception', None)

        @retry(wait_exponential_multiplier=wait_exponential_multiplier, wait_exponential_max=wait_exponential_max,
               stop_max_attempt_number=stop_max_attempt_number, retry_on_exception=retry_on_exception)
        def _f():
            return f(*args, **kwargs)

        return _f()

    @staticmethod
    def get_retry_on_exception(exc):
        """Retry based on exception raised from GET method

        :param exc:
        :return:
        """
        errors = ['ERR_AXDB_TABLE_NOT_FOUND', 'ERR_AXDB_INVALID_PARAM', AttributeError, TypeError, KeyError]
        return retry_on_errors(errors=errors, retry=False, caller=__name__)(exc)

    @staticmethod
    def create_retry_on_exception(exc):
        """Retry based on exception raised from CREATE method

        :param exc:
        :return:
        """
        errors = ['ERR_AXDB_TABLE_NOT_FOUND', 'ERR_AXDB_INSERT_DUPLICATE', 'ERR_AXDB_INVALID_PARAM']
        return retry_on_errors(errors=errors, retry=False, caller=__name__)(exc)

    @staticmethod
    def update_retry_on_exception(exc):
        """Retry based on exception raised from UPDATE method

        :param exc:
        :return:
        """
        errors = ['ERR_AXDB_TABLE_NOT_FOUND', 'ERR_AXDB_INVALID_PARAM', 'ERR_AXDB_CONDITIONAL_UPDATE_FAILURE']
        return retry_on_errors(errors=errors, retry=False, caller=__name__)(exc)

    @staticmethod
    def delete_retry_on_exception(exc):
        """Retry based on exception raised from DELETE method

        :param exc:
        :return:
        """
        errors = ['ERR_AXDB_TABLE_NOT_FOUND', 'ERR_AXDB_INVALID_PARAM']
        return retry_on_errors(errors=errors, retry=False, caller=__name__)(exc)

    @staticmethod
    def wait_for_table_exception(exc):
        """
        Retry till NOT_FOUND exception goes away

        :param exc:
        :return:
        """
        errors = ['ERR_AXDB_TABLE_NOT_FOUND']
        return retry_on_errors(errors=errors, retry=True, caller=__name__)(exc)


class AxdbClient(BaseAxdbClient):
    """AXDB client."""

    tables = {
        'workflow': '/axdevops/workflow',
        'workflow_leaf_service': '/axdevops/workflow_leaf_service',
        'workflow_events': '/axdevops/workflow_timed_events',
        'node_events': '/axdevops/workflow_node_event',
        'workflow_kv': '/axdevops/workflow_kv',
        'branch': '/axdevops/branch',
        'approval': '/axdevops/approval',
        'approval_result': '/axdevops/approval_result',
        'artifact': '/axsys/artifacts',
        'artifact_retention': '/axsys/artifact_retention',
        'resource': '/axdevops/resource',
    }

    conditional_update_columns = {
        'service': {
            'tags',
            'artifact_nums',
            'artifact_size'
        },
        'artifact': {
            'tags',
            'retention_tags'
        },
        'artifact_meta': {
            'value'
        },
        'resource': {
            'timestamp'
        }
    }

    def get_workflow_status(self, workflow_id):
        """Get workflow status.

        :param workflow_id:
        :return:
        """

        table_name = str(self.tables['workflow'])
        assert workflow_id, 'Missing workflow ID'

        try:
            response = self.retry_request('get', table_name, retry_on_exception=self.get_retry_on_exception, params={'id': workflow_id})
            workflow = response.json()
            if not workflow:
                return None
            assert len(workflow) == 1, "bad result with length != 1, id={} result={}".format(workflow_id, workflow)
            return workflow[0]
        except HTTPError as exc:
            logger.error('Error caused by %s', str(exc))
            raise exc
        except Exception as exc:
            logger.error('Error, %s', str(exc))
            raise exc

    def create_workflow_status(self, workflow_id, status, service_template, timestamp, resource):
        """Create a workflow.

        :param workflow_id:
        :param status:
        :param service_template:
        :param timestamp:
        :param resource:
        :return:
        """
        table_name = str(self.tables['workflow'])
        assert workflow_id and status, 'Missing workflow ID or Status'
        new_workflow = {
            'id': workflow_id,
            'status': status,
            'service_template': service_template,
            'timestamp': timestamp,
            'resource': resource
        }

        try:
            self.retry_request('post', table_name, retry_on_exception=self.create_retry_on_exception, data=new_workflow)
            return True
        except HTTPError as exc:
            try:
                data = exc.response.json()
                error_code = data.get('code', '')
                if error_code == 'ERR_AXDB_INSERT_DUPLICATE':
                    logger.warning('Key %s already exists in the table, fail to create. %s', workflow_id, exc.response.text)
                    return False
            except:
                pass
            logger.error('Error caused by %s', str(exc))
            raise exc
        except Exception as exc:
            logger.error('Error, %s', str(exc))
            raise exc

    def delete_workflow_status(self, workflow_id):
        """Delete a workflow.

        :param workflow_id:
        "return:
        """
        table_name = str(self.tables['workflow'])
        assert workflow_id, 'Missing workflow ID'

        try:
            self.retry_request('delete', table_name, data=[{'id': workflow_id}])
            return True
        except Exception as exc:
            logger.error('Error caused by %s', str(exc))
            raise exc

    def update_workflow_status(self, workflow_id, workflow):
        """Update a workflow.

        :param workflow_id:
        :param workflow:
        :return:
        """
        table_name = str(self.tables['workflow'])
        assert workflow_id, 'Missing workflow ID'
        assert ('status' in workflow or 'service_template' in workflow), \
            'Workflow object must have status and service_template attribute'
        new_workflow = {'id': workflow_id}

        if 'status' in workflow:
            new_workflow['status'] = workflow['status']
        if 'timestamp' in workflow:
            new_workflow['timestamp'] = workflow['timestamp']
        if 'service_template' in workflow:
            new_workflow['service_template'] = workflow['service_template']
        if 'resource' in workflow:
            new_workflow['resource'] = workflow['resource']

        try:
            self.retry_request('put', table_name, retry_on_exception=self.update_retry_on_exception, data=new_workflow)
            return True
        except Exception as exc:
            logger.error('Error caused by %s', str(exc))
            raise exc

    def update_conditional_workflow_status(self, workflow_id, timestamp, status, old_status):
        """Update an existed workflow status to status only if the old status is old_status.

        :param workflow_id:
        :param timestamp:
        :param status:
        :param old_status:
        :return:
        """
        table_name = str(self.tables['workflow'])
        assert workflow_id and status and old_status, 'Missing arguments'
        new_workflow = {'id': workflow_id, 'timestamp': timestamp, 'status': status, 'status_update_if': old_status}

        try:
            self.retry_request('put', table_name, retry_on_exception=self.update_retry_on_exception, data=new_workflow)
            return True
        except HTTPError as exc:
            # TODO: return False depends on error code AA-1052
            try:
                data = exc.response.json()
                error_msg = data.get('message', '')
                if error_msg.find('Conditional Update failed') != -1:
                    logger.warning('Conditional update failed. Fail to update. %s', exc.response.text)
                    raise exc
            except:
                pass
            logger.error('Error caused by %s', str(exc))
            raise exc
        except Exception as exc:
            logger.error('Error, %s', str(exc))
            raise exc

    def get_workflow_certain_columns(self, workflow_id, column_names):
        """Only get certain columns from workflow.

        :param workflow_id:
        :param column_names:
        :return:
        """
        table_name = str(self.tables['workflow'])
        assert workflow_id and column_names, 'Missing workflow ID or Column Name List'
        assert isinstance(column_names, list), 'Column Names has to ba a list'
        payload = {
            'id': workflow_id,
            'ax_select_cols': column_names
        }

        try:
            response = self.retry_request('get', table_name, retry_on_exception=self.get_retry_on_exception, params=payload)
            workflow = response.json()
            if not workflow:
                return None
            assert len(workflow) == 1, "bad result with length != 1, id={} cn={} result={}".format(workflow_id, column_names, workflow)
            return workflow[0]
        except HTTPError as exc:
            logger.error('Error caused by %s', str(exc))
            raise exc
        except Exception as exc:
            logger.error('Error, %s', str(exc))
            raise exc

    def get_workflows_certain_columns_by_status(self, status, column_names):
        """Only get certain columns from workflows that has specified status.

        :param status:
        :param column_names:
        :return:
        """
        table_name = str(self.tables['workflow'])
        assert status and column_names, 'Missing status or Column Name List'
        assert isinstance(column_names, list), 'Column Names has to ba a list'
        payload = {
            'status': status,
            'ax_select_cols': column_names
        }
        try:
            response = self.retry_request('get', table_name, retry_on_exception=self.get_retry_on_exception,
                                          params=payload)
            workflows = response.json()
            return workflows
        except HTTPError as exc:
            logger.error('Error caused by %s', str(exc))
            raise exc
        except Exception as exc:
            logger.error('Error, %s', str(exc))
            raise exc

    def get_leaf_service_result_by_leaf_id(self, leaf_id):
        """Get leaf service results with particular root id (workflow_id).

        :param leaf_id:
        :return:
        """
        table_name = str(self.tables['workflow_leaf_service'])
        assert leaf_id, 'Missing leaf service ID'

        try:
            response = self.retry_request('get', table_name, retry_on_exception=self.get_retry_on_exception, params={'leaf_id': leaf_id})
            leaf = response.json()
            return leaf
        except HTTPError as exc:
            logger.error('Error caused by %s', str(exc))
            raise exc
        except Exception as exc:
            logger.error('Error, %s', str(exc))
            raise exc

    def get_leaf_service_results(self, workflow_id):
        """Get leaf service results with particular root id (workflow_id).

        :param workflow_id:
        :return:
        """
        table_name = str(self.tables['workflow_leaf_service'])
        assert workflow_id, 'Missing workflow ID'

        try:
            response = self.retry_request('get', table_name, retry_on_exception=self.get_retry_on_exception, params={'root_id': workflow_id})
            leaf = response.json()
            assert isinstance(leaf, list), "bad result, expecting a list but gets {}".format(leaf)
            return leaf
        except HTTPError as exc:
            logger.error('Error caused by %s', str(exc))
            raise exc
        except Exception as exc:
            logger.error('Error, %s', str(exc))
            raise exc

    def create_leaf_service_result(self, leaf_id, root_id, sn, result, timestamp, detail=None):
        """Create a leaf service result.

        :param leaf_id:
        :param root_id:
        :param sn:
        :param result:
        :param timestamp
        :param detail:
        :return:
        """
        table_name = str(self.tables['workflow_leaf_service'])
        assert leaf_id and root_id and result and (type(sn) is int), 'Invalid argument for creating leaf service entry'
        new_leaf = {
            'leaf_id': leaf_id,
            'root_id': root_id,
            'sn': sn,
            'result': result,
            'timestamp': timestamp,
            'detail': detail
        }
        try:
            self.retry_request('post', table_name, retry_on_exception=self.create_retry_on_exception, data=new_leaf)
            return True
        except HTTPError as exc:
            try:
                data = exc.response.json()
                error_code = data.get('code', '')
                if error_code == 'ERR_AXDB_INSERT_DUPLICATE':
                    logger.warning('Key root_id=%s sn=%s already exists in the table, fail to create. leaf_id=%s %s',
                                   root_id, sn, leaf_id, exc.response.text)
                    return False
            except:
                pass
            logger.error('Error caused by %s', str(exc))
            raise exc
        except Exception as exc:
            logger.error('Error, %s', str(exc))
            raise exc

    def delete_leaf_service_result(self, leaf_id):
        """Delete a leaf service result.

        :param leaf_id:
        :return:
        """
        table_name = str(self.tables['workflow_leaf_service'])
        assert leaf_id, 'Missing Leaf Service Id'
        try:
            self.retry_request('delete', table_name, data=[{'leaf_id': leaf_id}])
            return True
        except Exception as exc:
            logger.error('Error caused by %s', str(exc))
            raise exc

    def get_workflow_events(self, workflow_id=None, timestamp_start=None):
        """Get workflow events with particular root id (workflow_id).

        :param workflow_id:
        :param timestamp_start:
        :return:
        """
        table_name = str(self.tables['workflow_events'])
        params = {}
        if workflow_id:
            params['root_id'] = workflow_id
        if timestamp_start:
            params[AxdbConstants.AXDBQueryMinTime] = timestamp_start

        try:
            response = self.retry_request('get', table_name, retry_on_exception=self.get_retry_on_exception,
                                          params=params)
            events = response.json()
            assert isinstance(events, list), "bad result, expecting a list but gets {}".format(events)
            return events
        except HTTPError as exc:
            logger.error('Error caused by %s', str(exc))
            raise exc
        except Exception as exc:
            logger.error('Error, %s', str(exc))
            raise exc

    def create_workflow_event(self, root_id, event_type, timestamp, detail=None):
        """Create a leaf service result.

        :param root_id:
        :param event_type:
        :param timestamp
        :param detail:
        :return:
        """
        table_name = str(self.tables['workflow_events'])
        assert event_type and root_id and timestamp and (type(timestamp) is int), 'Invalid argument for creating workflow event entry'
        new_event = {
            'root_id': root_id,
            'event_type': event_type,
            'timestamp': timestamp,
            'detail': detail
        }
        try:
            self.retry_request('post', table_name, retry_on_exception=self.create_retry_on_exception, data=new_event)
            return True
        except HTTPError as exc:
            try:
                data = exc.response.json()
                error_code = data.get('code', '')
                if error_code == 'ERR_AXDB_INSERT_DUPLICATE':
                    logger.warning('Key %s %s already exists in the table, fail to create. %s', root_id, timestamp, exc.response.text)
                    return False
            except:
                pass
            logger.error('Error caused by %s', str(exc))
            raise exc
        except Exception as exc:
            logger.error('Error, %s', str(exc))
            raise exc

    def get_node_events(self, root_id=None, leaf_id=None, timestamp_start=None):
        """Get node status reporting events."""
        table_name = str(self.tables['node_events'])
        params = {}
        if root_id:
            params['root_id'] = root_id
        if leaf_id:
            params['leaf_id'] = leaf_id
        if timestamp_start:
            params[AxdbConstants.AXDBQueryMinTime] = timestamp_start

        try:
            response = self.retry_request('get', table_name, retry_on_exception=self.get_retry_on_exception,
                                          params=params)
            events = response.json()
            assert isinstance(events, list), "bad result, expecting a list but gets {}".format(events)
            return events
        except HTTPError as exc:
            logger.error('Error caused by %s', str(exc))
            raise exc
        except Exception as exc:
            logger.error('Error, %s', str(exc))
            raise exc

    def create_node_event(self, leaf_id, root_id, result, timestamp, status_detail=None, detail=None):
        """Create a node event."""
        table_name = str(self.tables['node_events'])
        assert leaf_id and root_id and result, 'Invalid argument for creating leaf service entry'
        new_node_event = {
            'leaf_id': leaf_id,
            'root_id': root_id,
            'result': result,
            'timestamp': timestamp,
            'status_detail': status_detail,
            'detail': detail
        }
        try:
            self.retry_request('post', table_name, retry_on_exception=self.create_retry_on_exception, data=new_node_event)
            return True
        except HTTPError as exc:
            try:
                data = exc.response.json()
                error_code = data.get('code', '')
                if error_code == 'ERR_AXDB_INSERT_DUPLICATE':
                    logger.warning('Key root_id=%s leaf_id=%s already exists in the table, fail to create. %s',
                                   root_id, leaf_id, exc.response.text)
                    return False
            except:
                pass
            logger.error('Error caused by %s', str(exc))
            raise exc
        except Exception as exc:
            logger.error('Error, %s', str(exc))
            raise exc

    def get_workflow_kv(self, key):
        """Get value of key.

        :param key:
        :return: (value, timestamp) or None
        """
        table_name = str(self.tables['workflow_kv'])
        assert key, 'Missing key'

        try:
            response = self.retry_request('get', table_name, retry_on_exception=self.get_retry_on_exception,
                                          params={'key': key})
            l = response.json()
            assert isinstance(l, list)
            assert len(l) < 2
            if len(l):
                try:
                    value = l[0].get('value', '{}')
                    timestamp = l[0].get('timestamp', 0)
                    return value, timestamp
                except Exception:
                    raise
            else:
                return None, None
        except HTTPError as exc:
            logger.error('Error caused by %s', str(exc))
            raise exc
        except Exception as exc:
            logger.error('Error, %s', str(exc))
            raise exc

    def put_workflow_kv(self, key, value):
        """Create a leaf service result.

        :param key:
        :param value:
        :return:
        """
        table_name = str(self.tables['workflow_kv'])
        assert key, 'Invalid argument for creating workflow kv entry'
        assert isinstance(value, dict), 'bad value {}'.format(value)
        kv = {
            'key': key,
            'value': json.dumps(value),
            'timestamp': int(time.time() * 1000)
        }
        try:
            self.retry_request('put', table_name, retry_on_exception=self.create_retry_on_exception, data=kv)
            return True
        except HTTPError as exc:
            logger.error('Error caused by %s', str(exc))
            raise exc
        except Exception as exc:
            logger.error('Error, %s', str(exc))
            raise exc

    def get_branch_head(self, repo, branch):
        """Get head of branch.

        :param repo:
        :param branch:
        :return:
        """
        branch_heads = self.retry_function(self.query, self.tables['branch'], {'repo': repo, 'branch': branch},
                                           value_only=True, max_retry=2, retry_on_exception=self.delete_retry_on_exception)
        if branch_heads:
            return branch_heads[0]['head']

    def set_branch_head(self, repo, branch, head):
        """Set head of branch.

        :param repo:
        :param branch:
        :param head:
        :return:
        """
        return self.retry_function(self.update_entry, self.tables['branch'], {'repo': repo, 'branch': branch, 'head': head},
                                   value_only=True, max_retry=2, retry_on_exception=self.update_retry_on_exception)

    def purge_branch_head(self, repo, branch):
        """Purge branch head.

        :param repo:
        :param branch:
        :return:
        """
        return self.retry_function(self.delete_entry, self.tables['branch'], [{'repo': repo, 'branch': branch}],
                                   value_only=True, max_retry=2, retry_on_exception=self.delete_retry_on_exception)

    def get_branch_heads(self, repo):
        """Get all branch heads of a repo.

        :param repo:
        :return:
        """
        return self.retry_function(self.query, self.tables['branch'], {'repo': repo}, value_only=True,
                                   max_retry=2, retry_on_exception=self.delete_retry_on_exception)

    def purge_branch_heads(self, repo):
        """Purge all branch heads of a repo.

        :param repo:
        :return:
        """
        return self.retry_function(self.delete_entry, self.tables['branch'], [{'repo': repo}], value_only=True,
                                   max_retry=2, retry_on_exception=self.delete_retry_on_exception)

    def get_approval_info(self, root_id=None, leaf_id=None):
        """Get approval info."""
        table_name = str(self.tables['approval'])
        assert root_id or leaf_id, 'root_id and leaf_id cannot both be None'
        params = {}
        if root_id:
            params['root_id'] = root_id
        if leaf_id:
            params['leaf_id'] = leaf_id
        try:
            response = self.retry_request('get', table_name, retry_on_exception=self.get_retry_on_exception,
                                          params=params)
            events = response.json()
            assert isinstance(events, list), "bad result, expecting a list but gets {}".format(events)
            return events
        except HTTPError as exc:
            logger.error('Error caused by %s', str(exc))
            raise exc
        except Exception as exc:
            logger.error('Error, %s', str(exc))
            raise exc

    def create_approval_info(self, root_id, leaf_id, required_list, optional_list, optional_number, timeout, result, detail=None):
        """Create an approval info entry."""
        table_name = str(self.tables['approval'])
        assert leaf_id and root_id and result, 'Invalid argument for creating leaf service entry'
        new_approval_info = {
            'leaf_id': leaf_id,
            'root_id': root_id,
            'required_list': required_list,
            'optional_list': optional_list,
            'optional_number': optional_number,
            'timeout': timeout,
            'result': result,
        }
        if detail:
            new_approval_info['detail'] = detail
        try:
            self.retry_request('post', table_name, retry_on_exception=self.create_retry_on_exception, data=new_approval_info)
            return True
        except HTTPError as exc:
            try:
                data = exc.response.json()
                error_code = data.get('code', '')
                if error_code == 'ERR_AXDB_INSERT_DUPLICATE':
                    logger.warning('Key root_id=%s leaf_id=%s already exists in the table, fail to create. %s',
                                   root_id, leaf_id, exc.response.text)
                    return False
            except:
                pass
            logger.error('Error caused by %s', str(exc))
            raise exc
        except Exception as exc:
            logger.error('Error, %s', str(exc))
            raise exc

    def update_approval_info(self, root_id, leaf_id, approval_result):
        """Update an approval info entry."""
        table_name = str(self.tables['approval'])
        assert root_id and leaf_id, 'Missing root_id or leaf_id'
        assert ('result' in approval_result), \
            'approval info object must have result attribute'
        new_approval_info = {'root_id': root_id,
                             'leaf_id': leaf_id}

        if 'result' in approval_result:
            new_approval_info['result'] = approval_result['result']
        if 'detail' in approval_result:
            new_approval_info['detail'] = approval_result['detail']

        try:
            self.retry_request('put', table_name, retry_on_exception=self.update_retry_on_exception, data=new_approval_info)
            return True
        except Exception as exc:
            logger.error('Error caused by %s', str(exc))
            raise exc

    def get_approval_results(self, leaf_id=None, user=None):
        """Get approval results for approval step."""
        table_name = str(self.tables['approval_result'])
        assert leaf_id or user, 'leaf_id and user cannot both be None'
        params = {}
        if leaf_id:
            params['leaf_id'] = leaf_id
        if user:
            params['user'] = user
        try:
            response = self.retry_request('get', table_name, retry_on_exception=self.get_retry_on_exception,
                                          params=params)
            events = response.json()
            assert isinstance(events, list), "bad result, expecting a list but gets {}".format(events)
            return events
        except HTTPError as exc:
            logger.error('Error caused by %s', str(exc))
            raise exc
        except Exception as exc:
            logger.error('Error, %s', str(exc))
            raise exc

    def create_approval_results(self, leaf_id, root_id, result, user, timestamp):
        """Create an approval result."""
        table_name = str(self.tables['approval_result'])
        assert leaf_id and root_id and result and user, 'Invalid argument for creating leaf service entry'
        new_approval_result = {
            'leaf_id': leaf_id,
            'root_id': root_id,
            'result': result,
            'user': user,
            'timestamp': timestamp,
        }
        try:
            self.retry_request('post', table_name, retry_on_exception=self.create_retry_on_exception, data=new_approval_result)
            return True
        except HTTPError as exc:
            try:
                data = exc.response.json()
                error_code = data.get('code', '')
                if error_code == 'ERR_AXDB_INSERT_DUPLICATE':
                    logger.warning('Key root_id=%s leaf_id=%s already exists in the table, fail to create. %s',
                                   root_id, leaf_id, exc.response.text)
                    return False
            except Exception:
                pass
            logger.error('Error caused by %s', str(exc))
            raise exc
        except Exception as exc:
            logger.error('Error, %s', str(exc))
            raise exc

    def get_artifacts(self, params):
        """Search artifacts

        :param params:
        :returns:
        """
        return self.retry_function(self.query, self.tables['artifact'], params, value_only=True,
                                   max_retry=2, retry_on_exception=self.get_retry_on_exception)

    def create_artifact(self, payload):
        """Create an artifact

        :param payload:
        :returns:
        """
        return self.retry_function(self.create_entry, self.tables['artifact'], payload, value_only=True,
                                   max_retry=2, retry_on_exception=self.create_retry_on_exception)

    def update_artifact(self, payload):
        """Update an artifact

        :param payload:
        :returns:
        """
        return self.retry_function(self.update_entry, self.tables['artifact'], payload, value_only=True,
                                   max_retry=2, retry_on_exception=self.update_retry_on_exception)

    def update_artifact_conditionally(self, artifact_id, ax_uuid, **kwargs):
        """Update artifact if condition holds

        :param artifact_id:
        :param ax_uuid:
        :param kwargs:
        :returns:
        """
        params = self._create_conditional_update_payload(self.conditional_update_columns['artifact'], **kwargs)
        params['artifact_id'] = artifact_id
        params['ax_uuid'] = ax_uuid
        return self.update_artifact(params)

    def get_retention_policies(self, tag_name=None):
        """Get artifact retention tags with policy from retention table

        :param tag_name: string
        :returns:
        """
        params = {}
        if tag_name:
            params['name'] = tag_name
        return self.retry_request('get', str(self.tables['artifact_retention']), params=params, value_only=True,
                                  max_retry=2, retry_on_exception=self.get_retry_on_exception)

    def create_retention_policy(self, tag_name, policy, description=None):
        """Create an artifact retention policy based on tag name

        :param tag_name: string
        :param policy: integer
        :param description: string
        :returns:
        """
        new_retention = {
            'name': tag_name,
            'policy': policy,
        }
        if description:
            new_retention['description'] = description
        return self.retry_request('post', str(self.tables['artifact_retention']), data=new_retention, value_only=True,
                                  max_retry=2, retry_on_exception=self.create_retry_on_exception)

    def update_retention_policy(self, tag_name, policy, description):
        """Update an artifact retention policy based on tag name

        :param tag_name:
        :param policy:
        :param description:
        :returns:
        """
        new_retention = {
            'name': tag_name,
        }
        if policy:
            new_retention['policy'] = policy
        if description:
            new_retention['description'] = description
        return self.retry_request('put', str(self.tables['artifact_retention']), data=new_retention, value_only=True,
                                  max_retry=2, retry_on_exception=self.update_retry_on_exception)

    def update_retention_policy_metadata(self, tag_name, total_number, total_size, total_real_size):
        """Update an artifact retention policy meta data based on tag name

        :param tag_name:
        :param total_number:
        :param total_size:
        :param total_real_size:
        :return:
        """
        new_retention = {
            'name': tag_name,
            'total_number': total_number,
            'total_size': total_size,
            'total_real_size': total_real_size,
        }
        return self.retry_request('put', str(self.tables['artifact_retention']), data=new_retention, value_only=True,
                                  max_retry=2, retry_on_exception=self.update_retry_on_exception)

    def update_retention_policy_meta_conditionally(self, old_retention_policy, new_retention_policy):
        """Update a retention policy based conditionally
        :param old_retention_policy:
        :param new_retention_policy:
        :return:
        """
        assert old_retention_policy['name'] == new_retention_policy['name']
        new_retention = dict()
        new_retention['name'] = new_retention_policy['name']
        new_retention['total_number'] = new_retention_policy['total_number']
        new_retention['total_size'] = new_retention_policy['total_size']
        new_retention['total_real_size'] = new_retention_policy['total_real_size']

        new_retention['total_number_update_if'] = old_retention_policy['total_number']
        new_retention['total_size_update_if'] = old_retention_policy['total_size']
        new_retention['total_real_size_update_if'] = old_retention_policy['total_real_size']

        return self.retry_request('put', str(self.tables['artifact_retention']), data=new_retention, value_only=True,
                                  max_retry=2, retry_on_exception=self.update_retry_on_exception)

    def delete_retention_policy(self, tag_name):
        """Delete an artifact retention policy based on tag name

        :param tag_name:
        :returns:
        """
        return self.retry_request('delete', str(self.tables['artifact_retention']), data=[{'name': tag_name}], value_only=True,
                                  max_retry=2, retry_on_exception=self.update_retry_on_exception)

    def get_live_workflow(self):
        """Get the list of live workflow

        :return:
        """
        params = {'ax_select_cols': 'task_id', 'is_task': True}
        return self.retry_function(self.query, '/axops/live_service', params, value_only=True,
                                   max_retry=2, retry_on_exception=self.get_retry_on_exception)

    def get_done_workflow(self, params):
        """Get the list of done workflows
        
        :return:
        """

        return self.retry_function(self.query, '/axops/done_service', params, value_only=True,
                                   max_retry=2, retry_on_exception=self.get_retry_on_exception)

    def get_service(self, service_id):
        """Search for live/done services with given service ID

        :param service_id:
        :returns:
        """
        services = self.retry_function(self.query, '/axops/live_service', {'ax_uuid': service_id}, value_only=True,
                                       max_retry=2, retry_on_exception=self.get_retry_on_exception)
        if not services:
            services = self.retry_function(self.query, '/axops/done_service', {'ax_uuid': service_id}, value_only=True,
                                           max_retry=2, retry_on_exception=self.get_retry_on_exception)
        if services:
            return services[0]

    def update_service(self, service_id, payload):
        """Update live/done service

        :param service_id:
        :param payload:
        :returns:
        """
        errors_not_retry = [
            'ERR_AXDB_TABLE_NOT_FOUND',
            'ERR_AXDB_INVALID_PARAM',
            'ERR_AXDB_CONDITIONAL_UPDATE_FAILURE',
            'ERR_AXDB_CONDITIONAL_UPDATE_FAILURE_NOT_EXIST'
        ]
        payload['ax_uuid'] = service_id
        try:
            return self.retry_function(self.update_entry, '/axops/live_service', payload, value_only=True,
                                       max_retry=2, retry_on_exception=retry_on_errors(errors=errors_not_retry, retry=False))
        except HTTPError as e:
            error = e.response.json()
            error_code = error.get('code')
            if error_code == 'ERR_AXDB_CONDITIONAL_UPDATE_FAILURE_NOT_EXIST':
                # If we cannot find service in live service table, it must be in done service table or not existed
                return self.retry_function(self.update_entry, '/axops/done_service', payload, value_only=True,
                                           max_retry=2, retry_on_exception=retry_on_errors(errors=errors_not_retry, retry=False))
            raise

    def update_service_conditionally(self, service_id, template_name, **kwargs):
        """Update service conditionally

        :param service_id:
        :param template_name:
        :param kwargs:
        :returns:
        """
        params = self._create_conditional_update_payload(self.conditional_update_columns['service'], **kwargs)
        params['ax_uuid'] = service_id
        params['template_name'] = template_name
        return self.update_service(service_id, params)

    def get_artifact_meta(self, attribute):
        """Get metadata of artifact

        :param attribute:
        :returns:
        """
        metadata = self.retry_function(self.query, '/axsys/artifact_meta', {'attribute': attribute}, value_only=True,
                                       max_retry=2, retry_on_exception=self.get_retry_on_exception)
        if metadata:
            return metadata[0]

    def update_artifact_meta(self, attribute, value):
        params = {
            'attribute': attribute,
            'value': value,
        }
        return self.retry_function(self.update_entry, '/axsys/artifact_meta', params, value_only=True,
                                   max_retry=2, retry_on_exception=self.update_retry_on_exception)

    def update_artifact_meta_conditionally(self, attribute, value, condition):
        """Update metadata of artifact conditionally

        :param attribute:
        :param value:
        :param condition:
        :returns:
        """
        params = self._create_conditional_update_payload(
            self.conditional_update_columns['artifact_meta'], value=value, value_update_if=condition)
        params['attribute'] = attribute
        return self.retry_function(self.update_entry, '/axsys/artifact_meta', params, value_only=True,
                                   max_retry=2, retry_on_exception=self.update_retry_on_exception)

    @staticmethod
    def _create_conditional_update_payload(columns, **kwargs):
        """Create conditional update payload

        :param columns: the list of columns where conditional update is supported.
        :param kwargs:
        :returns:
        """
        params = {}
        for column in columns:
            if column not in kwargs:
                continue
            else:
                condition_key = column + '_update_if'
                if condition_key not in kwargs:
                    message = 'Conditional update needs condition'
                    detail = 'Conditional update on column ({}) needs condition ({})'.format(column, condition_key)
                    raise AXIllegalArgumentException(message, detail)
                params[column] = kwargs[column]
                params[condition_key] = kwargs[condition_key]
        return params

    def get_resources(self, params):
        """Get resources

        :param params:
        :returns:
        """
        return self.retry_function(self.query, self.tables['resource'], params, value_only=True,
                                   max_retry=5, retry_on_exception=self.get_retry_on_exception)

    def create_resource(self, payload):
        """Create a resource

        :param payload:
        :returns:
        """
        return self.retry_function(self.create_entry, self.tables['resource'], payload, value_only=True,
                                   max_retry=5, retry_on_exception=self.create_retry_on_exception)

    def update_resource(self, payload):
        """Update a resource

        :param payload:
        :returns:
        """
        return self.retry_function(self.update_entry, self.tables['resource'], payload, value_only=True,
                                   max_retry=5, retry_on_exception=self.update_retry_on_exception)

    def update_resource_conditionally(self, payload):
        """Update a resource conditionally

        :param payload:
        :returns:
        """
        assert 'timestamp_update_if' in payload
        return self.retry_function(self.update_entry, self.tables['resource'], payload, value_only=True,
                                   max_retry=5, retry_on_exception=self.update_retry_on_exception)

    def delete_resource(self, payload):
        """Delete a resource

        :param payload:
        :returns:
        """
        return self.retry_function(self.delete_entry, self.tables['resource'], [payload], value_only=True,
                                   max_retry=5, retry_on_exception=self.delete_retry_on_exception)

    def get_storage_class_by_name(self, name):
        """Get a storage class by its name"""
        params = {'name': name}
        storage_classes = self.retry_request('get', '/axops/storage_classes', params=params, max_retry=5, retry_on_exception=self.get_retry_on_exception, value_only=True)
        if not storage_classes:
            return None
        if len(storage_classes) > 1:
            raise AXException("Found multiple storage classes with name: {}".format(name))
        return storage_classes[0]

    def create_volume(self, volume):
        """Create a volume"""
        return self.retry_request('post', '/axops/volumes', data=volume, max_retry=5, retry_on_exception=self.create_retry_on_exception, value_only=True)

    def get_volumes(self, params=None):
        """Retrieve a list of volume filted by params"""
        return self.retry_request('get', '/axops/volumes', params=params, max_retry=5, retry_on_exception=self.get_retry_on_exception, value_only=True)

    def get_volume(self, volume_id):
        """Retrieve a volume by its id"""
        volumes = self.get_volumes(params={'id': volume_id})
        if len(volumes) == 0:
            return None
        if len(volumes) > 1:
            raise AXException("Found multiple volumes with id: {}".format(volume_id))
        return volumes[0]

    def get_volume_by_axrn(self, axrn):
        """Retreieve a volume by its axrn"""
        volumes = self.get_volumes(params={'axrn': axrn.lower()})
        if len(volumes) == 0:
            return None
        if len(volumes) > 1:
            raise AXException("Found multiple volumes with axrn: {}".format(axrn))
        return volumes[0]

    def update_volume(self, volume):
        """Retrieve a volume by its id"""
        if not volume.get('id'):
            raise AXIllegalArgumentException("Volume id required for updates")
        # Prevent upsert behavior of axdb
        volume['ax_update_if_exist'] = ""
        return self.retry_request('put', '/axops/volumes', data=volume, max_retry=5, retry_on_exception=self.update_retry_on_exception, value_only=True)

    def delete_volume(self, volume_id):
        """Delete a volume by its id"""
        return self.retry_request('delete', '/axops/volumes', data=[{'id': volume_id}], max_retry=5, retry_on_exception=self.delete_retry_on_exception, value_only=True)

    def get_fixture_requests(self, params=None):
        """Retrieve fixture requests"""
        return self.retry_request('get', '/axops/fixture_requests', params=params, max_retry=5, retry_on_exception=self.get_retry_on_exception, value_only=True)

    def get_fixture_request(self, service_id):
        """Get the fixture request by service_id
        :return: fixture_request if it was still in the queue"""
        requests = self.get_fixture_requests(params={'service_id': service_id})
        if len(requests) == 0:
            return None
        if len(requests) > 1:
            raise AXException("Found multiple requests with id: {}".format(service_id))
        return requests[0]

    def create_fixture_request(self, request):
        """Create a fixture request"""
        return self.retry_request('post', '/axops/fixture_requests', data=request, max_retry=5, retry_on_exception=self.create_retry_on_exception, value_only=True)

    def update_fixture_request(self, request):
        """Updates a fixture request"""
        if not request.get('service_id'):
            raise AXIllegalArgumentException("Service id required for updates")
        # Prevent upsert behavior of axdb
        request['ax_update_if_exist'] = ""
        return self.retry_request('put', '/axops/fixture_requests', data=request, max_retry=5, retry_on_exception=self.update_retry_on_exception, value_only=True)

    def delete_fixture_request(self, service_id):
        """Delete fixture request from request database"""
        return self.retry_request('delete', '/axops/fixture_requests', data=[{'service_id': service_id}], max_retry=5, retry_on_exception=self.delete_retry_on_exception, value_only=True)

    def get_fixture_classes(self, params=None):
        """Retrieve list of classes filtered by params"""
        return self.retry_request('get', '/axops/fixture_classes', params=params, max_retry=5, retry_on_exception=self.get_retry_on_exception, value_only=True)

    def get_fixture_class_by_name(self, name):
        """Retrieve a fixture class by its name"""
        classes = self.get_fixture_classes(params={'name': name})
        if len(classes) == 0:
            return None
        if len(classes) > 1:
            raise AXException("Found multiple classes with name: {}".format(name))
        return classes[0]

    def get_fixture_class_by_id(self, class_id):
        """Retreieve a fixture class by class id"""
        classes = self.get_fixture_classes(params={'id': class_id})
        if len(classes) == 0:
            return None
        if len(classes) > 1:
            raise AXException("Found multiple classes with id: {}".format(class_id))
        return classes[0]

    def create_fixture_class(self, fix_class):
        """Inserts a fixture class into database"""
        return self.retry_request('post', '/axops/fixture_classes', data=fix_class, max_retry=5, retry_on_exception=self.create_retry_on_exception, value_only=True)

    def update_fixture_class(self, fix_class):
        """Updates an existing fixture class into database"""
        if not fix_class.get('id'):
            raise AXIllegalArgumentException("Class id required for updates")
        fix_class = copy.deepcopy(fix_class)
        # Prevent upsert behavior of axdb
        fix_class['ax_update_if_exist'] = ""
        return self.retry_request('put', '/axops/fixture_classes', data=fix_class, max_retry=5, retry_on_exception=self.update_retry_on_exception, value_only=True)

    def delete_fixture_class(self, class_id):
        """Delete fixture class from database"""
        return self.retry_request('delete', '/axops/fixture_classes', data=[{'id': class_id}], max_retry=5, retry_on_exception=self.delete_retry_on_exception, value_only=True)

    def create_fixture_instance(self, instance):
        """Inserts a fixture instance into database"""
        return self.retry_request('post', '/axops/fixture_instances', data=instance, max_retry=5, retry_on_exception=self.create_retry_on_exception, value_only=True)

    def get_fixture_instances(self, params=None):
        """Inserts a fixture instance into database"""
        return self.retry_request('get', '/axops/fixture_instances', params=params, max_retry=5, retry_on_exception=self.get_retry_on_exception, value_only=True)

    def get_fixture_instance_by_id(self, instance_id):
        """Retreieve an instance by its id"""
        instances = self.get_fixture_instances(params={'id': instance_id})
        if len(instances) == 0:
            return None
        if len(instances) > 1:
            raise AXException("Found multiple instances with id: {}".format(instance_id))
        return instances[0]

    def update_fixture_instance(self, instance):
        """Updates an existing instance into database"""
        if not instance.get('id'):
            raise AXIllegalArgumentException("Fixture id required for updates")
        instance = copy.deepcopy(instance)
        # Prevent upsert behavior of axdb
        instance['ax_update_if_exist'] = ""
        return self.retry_request('put', '/axops/fixture_instances', data=instance, max_retry=5, retry_on_exception=self.update_retry_on_exception, value_only=True)

    def delete_fixture_instance(self, instance_id):
        """Delete instance from database"""
        return self.retry_request('delete', '/axops/fixture_instances', data=[{'id': instance_id}], max_retry=5, retry_on_exception=self.delete_retry_on_exception, value_only=True)

