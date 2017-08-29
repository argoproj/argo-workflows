#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2016 Applatix, Inc. All rights reserved.
#

"""
Module for AXWorkflow
"""

import json
import uuid
import time

from ax.devops.axdb.axdb_client import AxdbClient
from ax.exceptions import AXIllegalArgumentException


class AXWorkflow(object):
    SUSPENDED = 'SUSPENDED'
    ADMITTED = 'ADMITTED'
    ADMITTED_DEL = 'ADMITTED_DEL'
    RUNNING = 'RUNNING'
    RUNNING_DEL = 'RUNNING_DEL'
    RUNNING_DEL_FORCE = 'RUNNING_DEL_FORCE'
    DELETED = 'DELETED'
    SUCCEED = 'SUCCEED'
    FAILED = 'FAILED'
    FORCED_FAILED = 'FORCED_FAILED'

    REDIS_LIST_EXPIRE_SECONDS = 3600 * 24 * 1  # 1 days
    REDIS_QUERY_LIST_KEY = 'query-list-key-{}'
    REDIS_DEL_LIST_KEY = 'del-list-key-{}'
    REDIS_DEL_FORCE_LIST_KEY = 'del-force-list-key-{}'
    WFL_RESULT_KEY = 'result-key-{}'
    REDIS_RESULT_LIST_KEY = 'result-list-key-{}'
    WFL_LAUNCH_KEY = 'launch-key-{}'
    REDIS_LAUNCH_LIST_KEY = 'launch-list-key-{}'
    WFL_LAUNCH_ACK_KEY = 'launch-ack-key-{}'
    REDIS_LAUNCH_ACK_LIST_KEY = 'launch-ack-list-key-{}'
    REDIS_FIXTURE_TERMINATION_LIST_KEY = 'fixture-termination-list-{}'
    REDIS_FIXTURE_ASSIGNMENT_KEY = 'assignment:{}'
    REDIS_FIXTURE_ASSIGNMENT_LIST_KEY = 'notification:{}'
    REDIS_DEPLOYMENT_UP_KEY = 'deployment-up-key-{}'
    REDIS_DEPLOYMENT_UP_LIST_KEY = 'deployment-up-list-key-{}'

    tag_test_ax_workflow = "test_ax_workflow"
    tag_test_ax_workflow_executor_crash_second = "test_ax_workflow_executor_crash_second"
    tag_test_ax_workflow_expect_failure_leaf_node = "test_ax_workflow_expect_failure_leaf_node"

    def __init__(self, workflow_id, service_template, status=SUSPENDED, timestamp=None, resource=None, leaf_resource=None, sn=0):
        super(AXWorkflow, self).__init__()
        self._workflow_id = workflow_id
        if service_template is not None:
            self._service_template = service_template
        self._status = status
        self._timestamp = timestamp
        self._resource = AXWorkflowResource(resource)
        self._leaf_resource = AXWorkflowResource(leaf_resource)
        self._sn = sn

    def __eq__(self, other):
        return self.id == other.id and self.service_template == other.service_template

    def __str__(self):
        return "[{}]: status={} resource={} leaf_resource={}".format(self._workflow_id, self.status, self.resource, self.leaf_resource)

    @property
    def id(self):
        return self._workflow_id

    @property
    def status(self):
        return self._status

    @property
    def service_template(self):
        return self._service_template

    @property
    def resource(self):
        return self._resource

    @resource.setter
    def resource(self, resource):
        self._resource = resource

    @property
    def leaf_resource(self):
        return self._leaf_resource

    @leaf_resource.setter
    def leaf_resource(self, leaf_resource):
        self._leaf_resource = leaf_resource

    @property
    def sn(self):
        return self._sn

    @property
    def timestamp(self):
        return self._timestamp

    def free_template(self):
        self._service_template = None

    def set_status(self, status):
        self._status = status

    def set_timestamp(self, timestamp):
        # xxx todo: it might be updated and accessed from different threads
        self._timestamp = timestamp

    @staticmethod
    def get_db_version_wait_till_db_is_ready():
        max_retry = 360
        count = 0
        while True:
            count += 1
            try:
                version = AxdbClient().version
                return version
            except Exception:
                if count > max_retry:
                    raise
                time.sleep(10)

    @staticmethod
    def post_workflow_to_db(workflow):
        timestamp = AXWorkflow.get_current_epoch_timestamp_in_ms()
        workflow.set_timestamp(timestamp)
        resource_obj = {
            'resource': workflow.resource.resource,
            'leaf_resource': workflow.leaf_resource.resource
        }
        return AxdbClient().create_workflow_status(workflow_id=workflow.id,
                                                   status=workflow.status,
                                                   service_template=json.dumps(workflow.service_template),
                                                   timestamp=timestamp,
                                                   resource=json.dumps(resource_obj))

    @staticmethod
    def update_workflow_status_in_db(workflow, new_status):
        timestamp = AXWorkflow.get_current_epoch_timestamp_in_ms()
        ret = AxdbClient().update_conditional_workflow_status(workflow_id=workflow.id,
                                                              timestamp=timestamp,
                                                              status=new_status,
                                                              old_status=workflow.status)
        if ret:
            workflow.set_timestamp(timestamp)
        return ret

    @staticmethod
    def get_workflow_by_id_from_db(workflow_id, need_load_template=False):
        if need_load_template:
            result = AxdbClient().get_workflow_status(workflow_id)

            if result is None:
                return None
            else:
                resource = json.loads(result.get('resource', "{}") or "{}")
                return AXWorkflow(workflow_id=result.get('id', None),
                                  service_template=json.loads(result.get('service_template', {})),
                                  timestamp=result.get('timestamp', None),
                                  status=result.get('status', None),
                                  resource=resource.get('resource', None),
                                  leaf_resource=resource.get('leaf_resource', None))
        else:
            result = AxdbClient().get_workflow_certain_columns(workflow_id=workflow_id,
                                                               column_names=['id', 'status', 'timestamp', 'resource'])
            if result is None:
                return None
            else:
                resource = json.loads(result.get('resource', "{}") or "{}")
                return AXWorkflow(workflow_id=result.get('id', None),
                                  service_template=None,
                                  timestamp=result.get('timestamp', None),
                                  status=result.get('status', None),
                                  resource=resource.get('resource', None),
                                  leaf_resource=resource.get('leaf_resource', None))

    @staticmethod
    def get_workflows_by_status_from_db(status, need_load_template=False):
        column_names = ['id', 'status', 'timestamp', 'resource']
        if need_load_template:
            column_names.append('service_template')
        results = AxdbClient().get_workflows_certain_columns_by_status(status=status,
                                                                       column_names=column_names)
        final_res = []
        for result in results:
            resource = json.loads(result.get('resource', "{}") or "{}")
            final_res.append(AXWorkflow(workflow_id=result.get('id', None),
                                        service_template=result.get('service_template', {}),
                                        timestamp=result.get('timestamp', None),
                                        status=result.get('status', None),
                                        resource=resource.get('resource', None),
                                        leaf_resource=resource.get('leaf_resource', None)))
        return final_res

    @staticmethod
    def get_workflow_leaf_result(workflow_leaf_id):
        task_result_key = AXWorkflow.WFL_RESULT_KEY.format(workflow_leaf_id)
        return AxdbClient().get_workflow_kv(task_result_key)

    @staticmethod
    def get_current_epoch_timestamp_in_ms():
        return int(time.time() * 1000)

    @staticmethod
    def get_current_epoch_timestamp_in_sec():
        return int(time.time())

    @staticmethod
    def service_template_add_reporting_callback_param(service_template, instance_id, auto_retry, is_wfe):
        reporting_callback = {
            'uuid': instance_id,
            'cookie': {
                'start_timestamp': AXWorkflow.get_current_epoch_timestamp_in_ms(),
                'instance_salt': str(uuid.uuid4())
            },
            'run_once': not auto_retry,
        }

        reporting_callback['is_wfe'] = is_wfe

        if 'template' in service_template:
            if 'outputs' not in service_template['template']:
                service_template['template']['outputs'] = {}
            service_template['template']['outputs']['reporting_callback'] = reporting_callback


class AXWorkflowResource(object):
    RESOURCE_LIST = ["cpu_cores", "mem_mib"]

    def __init__(self, resource_list=None):

        self.resource = [0.0]*len(AXWorkflowResource.RESOURCE_LIST)
        if resource_list is not None:
            assert isinstance(resource_list, list), "Invalid resource initialization with non-list argument, {}.".format(resource_list)
            assert len(resource_list) is len(AXWorkflowResource.RESOURCE_LIST), "Invalid length for initialization argument, {}".format(resource_list)
            for i in range(len(AXWorkflowResource.RESOURCE_LIST)):
                if not isinstance(resource_list[i], float):
                    try:
                        self.resource[i] = float(resource_list[i])
                    except Exception:
                        raise Exception("Invalid resource with non-number argument, {}".format(resource_list))
                else:
                    self.resource[i] = resource_list[i]
                # assert resource_list[i] >= 0, "Invalid resource initialization with negative resource, {}".format(resource_list)

    def toJson(self):
        return json.dumps(self.resource)

    def __repr__(self):
        return str(self.resource)

    def __add__(self, other):
        assert isinstance(other, AXWorkflowResource), "Resource addition has to happen between same type."
        new_resource = [0.0]*len(AXWorkflowResource.RESOURCE_LIST)
        for i in range(len(AXWorkflowResource.RESOURCE_LIST)):
            new_resource[i] = self.resource[i] + other.resource[i]
        return AXWorkflowResource(new_resource)

    def __sub__(self, other):
        assert isinstance(other, AXWorkflowResource), "Resource subtraction has to happen between same type."
        new_resource = [0.0]*len(AXWorkflowResource.RESOURCE_LIST)
        for i in range(len(AXWorkflowResource.RESOURCE_LIST)):
            new_resource[i] = self.resource[i] - other.resource[i]
        return AXWorkflowResource(new_resource)

    def __lt__(self, other):
        assert isinstance(other, AXWorkflowResource), "Resource operation has to happen between same type."
        result = True
        for i in range(len(AXWorkflowResource.RESOURCE_LIST)):
            if not self.resource[i] < other.resource[i]:
                result = False
                break
        return result

    def __le__(self, other):
        assert isinstance(other, AXWorkflowResource), "Resource operation has to happen between same type."
        result = True
        for i in range(len(AXWorkflowResource.RESOURCE_LIST)):
            if not self.resource[i] <= other.resource[i]:
                result = False
                break
        return result

    def __gt__(self, other):
        assert isinstance(other, AXWorkflowResource), "Resource operation has to happen between same type."
        result = True
        for i in range(len(AXWorkflowResource.RESOURCE_LIST)):
            if not self.resource[i] > other.resource[i]:
                result = False
                break
        return result

    def __ge__(self, other):
        assert isinstance(other, AXWorkflowResource), "Resource operation has to happen between same type."
        result = True
        for i in range(len(AXWorkflowResource.RESOURCE_LIST)):
            if not self.resource[i] >= other.resource[i]:
                result = False
                break
        return result

    def __eq__(self, other):
        assert isinstance(other, AXWorkflowResource), "Resource operation has to happen between same type."
        result = True
        for i in range(len(AXWorkflowResource.RESOURCE_LIST)):
            if not abs(self.resource[i] - other.resource[i]) < 0.001:  # Almost equal
                result = False
                break
        return result

    def __ne__(self, other):
        return not self.__eq__(other)

    @staticmethod
    def find_max(resource1, resource2):
        assert isinstance(resource1, AXWorkflowResource), "Resource operation has to happen between same type."
        assert isinstance(resource2, AXWorkflowResource), "Resource operation has to happen between same type."
        max_resource = [0.0]*len(AXWorkflowResource.RESOURCE_LIST)
        for i in range(len(AXWorkflowResource.RESOURCE_LIST)):
            max_resource[i] = max(resource1.resource[i], resource2.resource[i])
        return AXWorkflowResource(max_resource)


class AXResource(object):
    def __init__(self, resource_id, category, resource, ttl, timestamp=None, detail=None):
        try:
            assert resource_id and category and resource and ttl, 'missing parameters'
            self.resource_id = resource_id
            self.category = category
            self.resource = AXWorkflowResource(resource)
            self.ttl = int(ttl)
            self.timestamp = timestamp
            self.detail = detail if detail else {}
        except Exception as e:
            raise AXIllegalArgumentException(str(e))

    def __repr__(self):
        resource_str = "Id: {}, Category: {}, Resource: {}, ttl: {}, detail: {}".\
            format(self.resource_id, self.category, json.dumps(self.resource), self.ttl, self.detail)
        return resource_str

    def toJson(self):
        return {
            'resource_id': self.resource_id,
            'category': self.category,
            'resource': self.resource.resource,
            'ttl': self.ttl,
            'timestamp': self.timestamp,
            'detail': str(self.detail),
        }

    @staticmethod
    def get_resource_from_payload(payload):
        resource_id = payload.get('resource_id', None)
        category = payload.get('category', None)
        ttl = payload.get('ttl', None)
        cpu_resource = payload.get('cpu_cores', None)
        memory_resource = payload.get('mem_mib', None)
        detail = payload.get('detail', None)
        resource = [cpu_resource, memory_resource]

        return AXResource(resource_id=resource_id, category=category, resource=resource, ttl=ttl, detail=detail)

    @staticmethod
    def get_resources_from_db(params=None):
        results = AxdbClient().get_resources(params=params)
        final_res = []
        for result in results:
            resource = json.loads(result.get('resource', "[]") or "[]")
            final_res.append(AXResource(resource_id=result.get('resource_id', None),
                                        category=result.get('category', None),
                                        resource=resource,
                                        ttl=result.get('ttl', None),
                                        timestamp=result.get('timestamp', None),
                                        detail=result.get('detail', None)))
        return final_res
