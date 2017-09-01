#str(!/usr/bin/env python)
# -*- coding: utf-8 -*-
#
# Copyright 2016 Applatix, Inc. All rights reserved.
#

"""
Module for AXWorkflowExecutor
"""

import copy
import json
import logging
import math
import os
import pprint
import random
import re
import requests
import threading
import time
import traceback
import uuid

from retrying import retry

from ax.version import __version__
from ax.exceptions import AXIllegalArgumentException
from ax.devops.axdb.axdb_client import AxdbClient
from ax.devops.axsys.axsys_client import AxsysClient
from ax.devops.kafka.kafka_client import ExecutorProducerClient
from ax.devops.redis.redis_client import RedisClient, REDIS_HOST, DB_RESULT
from .service_template_pre_process import service_template_pre_process
from .ax_workflow import AXWorkflow, AXWorkflowResource
from .ax_workflow_constants import INSTANCE_RESOURCE, MINIMUM_RESOURCE_SCALE

logger = logging.getLogger(__name__)

axsys_client = AxsysClient()
axdb_client = AxdbClient()
redis_client = RedisClient(host=REDIS_HOST, db=DB_RESULT, retry_max_attempt=360, retry_wait_fixed=5000)

global_test_mode = False

MAX_TERMINATION_DELETION_ISSUED = 10


class AXWorkflowEvent(object):
    START = "START"
    EXCEPTION = "EXCEPTION"
    TERMINATE = "TERMINATE"
    FORCE_START = "FORCE_START"
    FORCE_DELETE = "FORCE_DELETE"
    FORCE_TERMINATE = "FORCE_TERMINATE"

    def __init__(self, workflow_id, event_type, detail, timestamp=None):
        super(AXWorkflowEvent, self).__init__()
        self._workflow_id = workflow_id
        self._event_type = event_type
        if not detail:
            self._detail = {}
        else:
            self._detail = detail
        if timestamp:
            self._timestamp = timestamp
        else:
            self._timestamp = AXWorkflow.get_current_epoch_timestamp_in_ms()

        self.log_prefix = "[WFE] [{}]:".format(self._workflow_id)

    def jsonify(self, no_nodes_stats=False):
        detail = self._detail if self._detail else {}
        if no_nodes_stats:
            if "nodes_stats" in detail:
                detail = copy.deepcopy(detail)
                detail.pop("nodes_stats", None)
        return {
            "workflow_id": self._workflow_id,
            "event_type": self._event_type,
            "detail": detail,
            "timestamp": self.timestamp
        }

    @property
    def workflow_id(self):
        return self._workflow_id

    @property
    def event_type(self):
        return self._event_type

    @property
    def detail(self):
        return self._detail

    @property
    def timestamp(self):
        return self._timestamp

    @staticmethod
    def save_workflow_exception_event_to_db(workflow_id, exception):
        e = {"exception": str(exception),
             "backtrace": "{}".format(traceback.format_exc())}

        a = AXWorkflowEvent(workflow_id=workflow_id, detail=e, event_type=AXWorkflowEvent.EXCEPTION)
        a._save_workflow_event_to_db()

    @staticmethod
    def save_workflow_event_to_db(workflow_id, event_type, detail=None, timestamp=None):
        a = AXWorkflowEvent(workflow_id=workflow_id, detail=detail, event_type=event_type, timestamp=timestamp)
        a._save_workflow_event_to_db()

    @staticmethod
    def _save_workflow_event_to_db_helper(workflow_id, event_type, timestamp, detail):
        if detail:
            detail_string = json.dumps(detail)
        else:
            detail_string = json.dumps({})
        axdb_client.create_workflow_event(root_id=workflow_id, timestamp=timestamp,
                                          event_type=event_type,
                                          detail=detail_string)

    def _save_workflow_event_to_db(self):
        logger.error("%s save event %s", self.log_prefix, self.jsonify())
        self._save_workflow_event_to_db_helper(workflow_id=self.workflow_id,
                                               event_type=self.event_type,
                                               timestamp=self.timestamp,
                                               detail=self.detail)

    @staticmethod
    def load_events_from_db(workflow_id=None, timestamp_start=None):
        results = axdb_client.get_workflow_events(workflow_id=workflow_id, timestamp_start=timestamp_start)
        ret = []
        for r in results:
            r = AXWorkflowEvent(workflow_id=r.get("root_id", None),
                                event_type=r.get("event_type", None),
                                detail=json.loads(r.get("detail", "{}")),
                                timestamp=r.get("timestamp", None))
            ret.append(r)
        return sorted(ret, key=lambda result: result.timestamp)


class AXWorkflowNodeResult(object):
    """Represents Leaf Service Result in AXDB"""
    LAUNCHED = "LAUNCHED"
    INTERRUPTED = "INTERRUPTED"
    SUCCEED = "SUCCEED"
    FAILED = "FAILED"

    FAILED_NONE_ZERO_RETURN = "FAILED_NONE_ZERO_RETURN"
    FAILED_SIG_TERM = "FAILED_SIG_TERM"
    FAILED_CANNOT_FIND_RETURN = "FAILED_CANNOT_FIND_RETURN"
    FAILED_CANNOT_LOAD_ARTIFACT = "FAILED_CANNOT_LOAD_ARTIFACT"
    FAILED_CANNOT_SAVE_ARTIFACT = "FAILED_CANNOT_SAVE_ARTIFACT"
    FAILED_NOT_ALLOW_RETRY = "FAILED_NOT_ALLOW_RETRY"
    FAILED_CANNOT_LAUNCH_CONTAINER_DRY = "FAILED_CANNOT_LAUNCH_CONTAINER_DRY"
    FAILED_CANNOT_LAUNCH_CONTAINER = "FAILED_CANNOT_LAUNCH_CONTAINER"
    FAILED_CANNOT_PULL_IMAGE = "FAILED_CANNOT_PULL_IMAGE"
    FAILED_LOST_CONTAINER = "FAILED_LOST_CONTAINER"
    FAILED_BAD_TEMPLATE = "FAILED_BAD_TEMPLATE"
    FAILED_FORCE_TERMINATE = "FAILED_FORCE_TERMINATE"
    FAILED_OOM_KILLED = "INSUFFICIENT_MEMORY"
    FAILED_CANNOT_CONNECT_APPLICATION_MANAGER = "CANNOT_CONNECT_APPLICATION_MANAGER"

    DETAIL_TAG_FAILURE_REASON = "failure_reason"
    DETAIL_TAG_FAILURE_MESSAGE = "failure_message"
    DETAIL_TAG_CONTAINER_RETURN_JSON = "container_return_json"
    DETAIL_TAG_CONTAINER_RETURN_BAD = "container_return_bad"
    DETAIL_TAG_FIXTURE_TERMINATED_BEFORE_LAUNCH = "fixture_terminated_before_launch"
    DETAIL_TAG_FIXTURE_TERMINATED_BY_EXECUTOR = "fixture_terminated_by_executor"
    DETAIL_TAG_DEPLOYMENT_TERMINATED_BEFORE_LAUNCH = "fixture_deployment_before_launch"
    DETAIL_TAG_DEPLOYMENT_TERMINATED_BY_REQUEST = "deployment_terminated_by_request"

    def __init__(self, workflow_id, node_id, sn=-1, detail=None, result_code=None, timestamp=None):
        self.workflow_id = workflow_id
        self.node_id = node_id
        self.sn = sn
        self.result_code = result_code
        self.detail = detail
        self.timestamp = timestamp
        self.log_prefix = "[WFE] [{}]:".format(self.workflow_id)
        assert self.is_valid, "Invalid arguments to create a AXWorkflowNodeResult instance."

    def __repr__(self):
        return "Leaf node result: %s, Workflow id: %s, Node id: %s, Detail: %s. Timestamp: %s" % \
               (self.result_code, self.workflow_id, self.node_id, self.detail, self.timestamp)

    def jsonify(self):
        return {
            'workflow_id': self.workflow_id,
            'node_id': self.node_id,
            'sn': self.sn,
            'result_code': self.result_code,
            'detail': self.detail,
            'timestamp': self.timestamp
        }

    @property
    def is_valid(self):
        return self.workflow_id and self.node_id and isinstance(self.sn, int) and \
               (self.result_code in {self.LAUNCHED, self.INTERRUPTED, self.SUCCEED, self.FAILED})

    @property
    def is_interrupted(self):
        return self.result_code == AXWorkflowNodeResult.INTERRUPTED

    @property
    def is_launched(self):
        return self.result_code == AXWorkflowNodeResult.LAUNCHED

    @property
    def is_succeed(self):
        return self.result_code == AXWorkflowNodeResult.SUCCEED

    @property
    def is_failed(self):
        return self.result_code == AXWorkflowNodeResult.FAILED

    @property
    def can_be_saved(self):
        return self.sn >= 0 and self.is_valid

    def id_matched(self, workflow_id, node_id):
        return node_id == self.node_id and workflow_id == self.workflow_id

    @classmethod
    def create_instance(cls, result):
        """
        Return a AXWorkflowNodeResult instance from an axdb leaf service result.
        :param result: result fetched from axdb
        :return:
        """
        workflow_id = result.get('root_id', None)
        node_id = result.get('leaf_id', None)
        sn = result.get('sn', -1)
        detail = json.loads(result.get('detail', "{}"))
        result_code = result.get('result', None)
        timestamp = result.get('timestamp', None)
        return AXWorkflowNodeResult(workflow_id, node_id, sn, detail, result_code, timestamp)

    @staticmethod
    def save_result_to_db_helper(leaf_id, workflow_id, sn, result_code, timestamp, detail):
        if detail:
            detail_string = json.dumps(detail)
        else:
            detail_string = json.dumps({})
        assert axdb_client.create_leaf_service_result(leaf_id, workflow_id, sn, result_code, timestamp, detail_string)

    def save_result_to_db(self):
        assert self.can_be_saved, "{} cannot save result={}".format(self.log_prefix,
                                                                    self.jsonify())
        self.save_result_to_db_helper(self.node_id, self.workflow_id,
                                      self.sn, self.result_code,
                                      self.timestamp, self.detail)

    @staticmethod
    def load_results_from_db(workflow_id):
        results = axdb_client.get_leaf_service_results(workflow_id)
        ret = []
        for r in results:
            r = AXWorkflowNodeResult.create_instance(r)
            ret.append(r)
        return sorted(ret, key=lambda result: result.sn)

    @staticmethod
    def get_leaf_service_results_by_leaf_id_from_db(leaf_id):
        results = axdb_client.get_leaf_service_result_by_leaf_id(leaf_id=leaf_id)
        ret = []
        for r in results:
            r = AXWorkflowNodeResult.create_instance(r)
            ret.append(r)
        return sorted(ret, key=lambda result: result.sn)

    @staticmethod
    def factory_for_parent(workflow_id, node_id, result_code):
        # sn for none-leaf node is always -1 (so it won't be saved)
        return AXWorkflowNodeResult(workflow_id=workflow_id, node_id=node_id, sn=-1,
                                    result_code=result_code,
                                    timestamp=AXWorkflow.get_current_epoch_timestamp_in_ms())

    def get_node_event_code(self):
        if self.is_succeed:
            return ExecutorProducerClient.SUCCESS_RESULT
        elif self.is_failed:
            return ExecutorProducerClient.FAILURE_RESULT
        elif self.is_interrupted:
            return ExecutorProducerClient.CANCELLED_RESULT
        else:
            return ExecutorProducerClient.RUNNING_STATE


class WorkflowNode(object):
    """Base Workflow Node"""
    UNKNOWN_STATE = "UNKNOWN_STATE"
    FRESH_STATE = "FRESH_STATE"
    EXPECTING_STATE = "EXPECTING_STATE"
    LAUNCHED_STATE = "LAUNCHED_STATE"
    INTERRUPTED_STATE = "INTERRUPTED_STATE"
    SUCCEED_STATE = "SUCCEED_STATE"
    FAILED_STATE = "FAILED_STATE"

    ARTIFACT_RESOURCE = [0.001, 4.0]  # Resource consumption for artifact container

    def __init__(self, name, full_path, service_template, node_id, parent_node, executor, flags):
        self.name = name
        self.full_path = full_path
        self.node_id = node_id
        self.service_template = copy.deepcopy(service_template)
        self.parent_node = parent_node
        self.children_nodes = []
        self.fixtures_nodes = []
        self.executor = executor
        if isinstance(flags, dict):
            self.flag_ignore_error = flags.get('ignore_error', False)
            self.flag_auto_retry = flags.get('auto_retry', None)
            self.flag_always_run = flags.get('always_run', False)
            self.flag_skipped = flags.get('skipped', False)
        else:
            self.flag_ignore_error = False
            self.flag_auto_retry = None
            self.flag_always_run = False
            self.flag_skipped = False

        self.dry_run = self.flag_skipped or (parent_node and parent_node.dry_run)

        self.ignore_delete_interrupt = False

        self._state = WorkflowNode.FRESH_STATE
        self._state_lock = threading.Lock()
        self._service_name_lock = threading.Lock()
        self._service_name = None
        self.result = None
        self._resource = None
        self._max_resource = None
        self._max_leaf_resource = None

        self.is_fixture = False
        self._fixtures_termination_triggered_lock = threading.Lock()
        self._fixtures_termination_triggered = False
        self._fixtures_need_to_be_terminated = False
        self._launched_time = 0

        self.load_artifacts_timestamp = 0
        self.save_artifacts_timestamp = 0

        self._parameters_lock = threading.Lock()
        self._parameters = {}

        self.log_prefix = "[WFE] [{}] [{}] [{}]:".format(self.workflow_id, self.node_id, self.name)

    def __str__(self, level=0):
        ret = "    "*level + repr(self)+"\n"
        for fixture in self.fixtures_nodes:
            ret += fixture.__str__(level + 1)
        for child in self.children_nodes:
            ret += child.__str__(level + 1)
        return ret

    def __repr__(self):
        return '%s node: %s, Node id: %s, State: %s, Parent: %s, Resource: %s' % \
               (self.get_type_string(), self.name, self.node_id, self.state, self.get_parent_id(), self.max_resource)

    def jsonify(self):
        raise NotImplementedError

    def d3_format(self):
        data = dict()
        data['name'] = self.name
        data['state_code'] = self.state
        data['type'] = self.get_type_string()
        data['service_template'] = json.dumps(self.service_template, indent=2)
        if self.result:
            data['result_code'] = self.result.result_code
        else:
            data['result_code'] = "None"
        if self.parent_node:
            data['parent'] = "{}-{}".format(self.parent_node.get_type_string(), self.parent_node.node_id)
        else:
            data['parent'] = "null"
        if self.fixtures_nodes or self.children_nodes:
            data["children"] = []
            for node in self.fixtures_nodes + self.children_nodes:
                data["children"].append(node.d3_format())
        return data

    @property
    def state(self):
        with self._state_lock:
            return self._state

    @state.setter
    def state(self, new_state):
        with self._state_lock:
            self._state = new_state

    @property
    def service_name(self):
        with self._service_name_lock:
            return self._service_name

    @service_name.setter
    def service_name(self, new_service_name):
        with self._service_name_lock:
            self._service_name = new_service_name

    @property
    def workflow_id(self):
        if self.executor:
            return self.executor.workflow_id
        return None

    @property
    def is_root(self):
        return self.parent_node is None

    @property
    def is_leaf(self):
        return False

    @property
    def is_deployment(self):
        return False

    @property
    def is_dind(self):
        return False

    @property
    def is_fresh(self):
        return self.state == WorkflowNode.FRESH_STATE

    @property
    def is_expecting(self):
        return self.state == WorkflowNode.EXPECTING_STATE

    @property
    def is_launched(self):
        return self.state == WorkflowNode.LAUNCHED_STATE

    @property
    def is_interrupted(self):
        return self.state == WorkflowNode.INTERRUPTED_STATE

    @property
    def is_succeed(self):
        return self.state == WorkflowNode.SUCCEED_STATE

    @property
    def is_failed(self):
        return self.state == WorkflowNode.FAILED_STATE

    @property
    def is_expecting_or_launched(self):
        return self.state in {WorkflowNode.EXPECTING_STATE, WorkflowNode.LAUNCHED_STATE}

    @property
    def is_done(self):
        return self.state in {WorkflowNode.INTERRUPTED_STATE, WorkflowNode.FAILED_STATE, WorkflowNode.SUCCEED_STATE}

    @property
    def has_fixtures(self):
        return len(self.fixtures_nodes) > 0

    @property
    def fixtures_need_to_be_terminated(self):
        return self._fixtures_need_to_be_terminated

    @property
    def fixtures_termination_triggered(self):
        with self._fixtures_termination_triggered_lock:
            return self._fixtures_termination_triggered

    @property
    def max_resource(self):
        return self._get_max_resource()

    @property
    def max_leaf_resource(self):
        if not self._max_leaf_resource:
            self._max_leaf_resource = self._get_max_leaf_resource()
        return self._max_leaf_resource

    def start(self, is_recover=False, in_cleanup_mode=None):
        raise NotImplementedError

    def get_type_string(self):
        raise NotImplementedError

    def _get_max_resource(self):
        """The lowed bound for resource required to accommodate the node."""
        raise NotImplementedError

    def _get_max_leaf_resource(self):
        """The max for each resource category for all leaf nodes."""
        if isinstance(self, LeafWorkflowNode):
            return self.max_resource
        else:
            result = AXWorkflowResource()
            for node in self.fixtures_nodes + self.children_nodes:
                resource = node.max_leaf_resource
                result = AXWorkflowResource.find_max(result, resource)
            return result

    def get_fixtures_ids(self):
        if self.fixtures_nodes:
            return [x.node_id for x in self.fixtures_nodes]
        return []

    def get_children_ids(self):
        if self.children_nodes:
            return [x.node_id for x in self.children_nodes]
        return []

    def get_parent_id(self):
        if self.parent_node:
            return self.parent_node.node_id
        return None

    def get_service_template_name(self):
        try:
            return self.service_template['template']['name']
        except Exception:
            return None

    def process_result(self, result, is_recover):
        assert self.result is None or self.result.is_launched, "{} {} already processed result {} in={}"\
            .format(self.log_prefix, self._state, self.result, result)

        assert isinstance(result, AXWorkflowNodeResult)
        self.result = result
        logger.info("%s %s (sn=%s, result=%s, detail=%s)",
                    self.log_prefix, self.state, result.sn, result.result_code, result.detail)
        assert result.is_valid
        assert result.id_matched(workflow_id=self.workflow_id, node_id=self.node_id)
        assert not self.is_fresh

        if self.is_interrupted:
            assert result.is_interrupted
            return
        elif self.is_succeed:
            assert result.is_succeed
            return
        elif self.is_failed:
            assert result.is_failed
            return
        elif self.is_expecting:
            # can be any result
            pass
        elif self.is_launched:
            if result.is_launched:
                if isinstance(self, ParallelWorkflowNode):
                    # parallel node may get two launched results, one from first child , one from last child
                    pass
                else:
                    return
        else:
            assert False

        # must in expecting state here
        if result.is_interrupted:
            new_state = WorkflowNode.INTERRUPTED_STATE
        elif result.is_succeed:
            new_state = WorkflowNode.SUCCEED_STATE
        elif result.is_failed:
            new_state = WorkflowNode.FAILED_STATE
        elif result.is_launched:
            if not self.is_launched:
                self._launched_time = result.timestamp
            new_state = WorkflowNode.LAUNCHED_STATE
        else:
            assert False

        logger.info("%s %s->%s", self.log_prefix, self.state, new_state)
        self.state = new_state

        if result.is_launched and self.is_fixture and self.is_leaf:
            parameters = result.detail.get("output_parameters", {})
            if isinstance(parameters, dict):
                for name in parameters:
                    if isinstance(parameters[name], dict):
                        for param in parameters[name]:
                            self.parent_node.set_parameters(key="fixtures.{}.{}".format(name, param), value=parameters[name][param])

        if not is_recover:
            if self.is_leaf:
                self.report_result_to_kafka(result.timestamp)
                # save per leaf-node, i.e. container result to DB
                if not self.dry_run:
                    result.save_result_to_db()
            elif isinstance(self, SequentialWorkflowNode):
                self.report_result_to_kafka(result.timestamp)
            else:
                # parallel node are fake node, don't report status
                pass
        else:  # check recovery mode if the kafka event is sent out correctly before crash
            if not self.is_leaf and isinstance(self, SequentialWorkflowNode):
                if self.is_done and not self.executor.check_event_already_sent(leaf_id=self.node_id, status=result.get_node_event_code()):
                    logger.info("%s (Recover mode) Sending missing finished state to Kafka", self.log_prefix)
                    self.report_result_to_kafka(result.timestamp)

        if self.is_root:
            assert not self.is_fixture
            if self.is_done:
                self.executor.last_step(result_code=result.result_code, forced=False)
        else:
            self.parent_node.process_child_result(self, is_recover)

    def process_result_code(self, result_code, is_recover):
        result = AXWorkflowNodeResult.factory_for_parent(self.workflow_id, self.node_id, result_code)
        return self.process_result(result, is_recover=is_recover)

    def report_result_to_kafka(self, timestamp=None):
        logger.info("%s Report state %s to Kafka.", self.log_prefix, self.state)
        result = self.get_report_result(timestamp)

        if result:
            self.executor.do_report_to_kafka(self.node_id, result)
        else:
            logger.info("%s state %s should not report to Kafka.", self.log_prefix, self.state)

    def get_report_result(self, timestamp):
        if timestamp is None:
            timestamp = AXWorkflow.get_current_epoch_timestamp_in_ms()
        states = dict()
        states['service_id'] = self.node_id
        states['root_id'] = self.workflow_id
        if self.is_expecting:
            states['status'] = ExecutorProducerClient.WAITING_STATE
            if self.has_fixtures:
                detail = "SETUP_FIXTURE"
            elif self.load_artifacts_timestamp:
                timestamp = self.load_artifacts_timestamp
                detail = "LOADING_ARTIFACTS"
            else:
                if hasattr(self, "first_time_see_image_pull") and self.first_time_see_image_pull:
                    detail = "CONTAINER_IMAGE_PULL_BACKOFF"
                else:
                    detail = "WAITING_FOR_RESOURCE"
            states['status_detail'] = {'code': detail}
            states['start_date'] = timestamp
        elif self.is_launched:
            states['status'] = ExecutorProducerClient.RUNNING_STATE
            if self.save_artifacts_timestamp:
                timestamp = self.save_artifacts_timestamp
                detail = "SAVING_ARTIFACTS"
            else:
                detail = "CONTAINER_RUNNING"
            states['status_detail'] = {'code': detail}
            if isinstance(self, StaticFixtureNode):
                d = self.result.detail
                assert "output_parameters" in d and "to_fixture_manager" in d, \
                    "{} invalid fixture result {}".format(self.log_prefix, self.result)
                states["static_fixture_parameter"] = d["output_parameters"]
                states["to_fixture_manager"] = d["to_fixture_manager"]
                states['status_detail'] = {'code': "FIXTURE_RUNNING"}
            states['start_date'] = timestamp
        elif self.is_done:
            if self.flag_skipped:
                states['status'] = ExecutorProducerClient.SKIPPED_RESULT
                states['status_detail'] = {'code': "TASK_SKIPPED"}
            elif self.is_succeed:
                states['status'] = ExecutorProducerClient.SUCCESS_RESULT
                states['status_detail'] = {'code': "TASK_SUCCEED"}
            elif self.is_interrupted:
                states['status'] = ExecutorProducerClient.CANCELLED_RESULT
                states['status_detail'] = {'code': "TASK_CANCELLED"}
            else:
                states['status'] = ExecutorProducerClient.FAILURE_RESULT
                states['status_detail'] = {'code': "TASK_FAILED"}
            start_date = self._launched_time
            if not start_date or start_date <= 0 or timestamp < start_date:
                run_duration = 0
            else:
                run_duration = timestamp - start_date
            states['run_duration'] = run_duration
            states['end_date'] = timestamp
            if self.result and self.result.detail:
                states["detail"] = self.result.detail
                if AXWorkflowNodeResult.DETAIL_TAG_FAILURE_REASON in self.result.detail:
                    states['status_detail'] = {'code': self.result.detail[AXWorkflowNodeResult.DETAIL_TAG_FAILURE_REASON]}
                    if AXWorkflowNodeResult.DETAIL_TAG_FAILURE_MESSAGE in self.result.detail:
                        states['status_detail']['message'] = self.result.detail[AXWorkflowNodeResult.DETAIL_TAG_FAILURE_MESSAGE]

        if isinstance(self, DeploymentNode):
            if self.result and self.result.detail:
                states['status_detail'] = self.result.detail
        return states

    def terminate_all_fixtures(self, is_recover):
        with self._fixtures_termination_triggered_lock:
            self._fixtures_termination_triggered = True

        no_pending = True
        for child in self.fixtures_nodes:
            if child.is_expecting_or_launched:
                logger.info("%s terminate %s", self.log_prefix, child.node_id)
                if not child.terminate_all_fixtures(is_recover):
                    no_pending = False
            elif child.is_fresh or child.is_done:
                pass
            else:
                assert False
        logger.info("%s termination return %s", self.log_prefix, no_pending)
        return no_pending

    def set_parameters(self, key, value):
        if isinstance(self, SequentialWorkflowNode) or self.is_root:
            with self._parameters_lock:
                try:
                    v = str(value)
                    self._parameters[key] = v
                    logger.info("%s set parameters ['%s'] to %s", self.log_prefix, key, v)
                except Exception:
                    logger.exception("%s exception in set_parameter", self.log_prefix)
        else:
            logger.info("%s set parent parameters", self.log_prefix)
            self.parent_node.set_parameters(key, value)

    def get_parent_deepcopied_parameters(self):
        if self.is_root:
            return {}
        else:
            return self.parent_node.get_deepcopied_parameters()

    def get_deepcopied_parameters(self):
        with self._parameters_lock:
            my_parameter = copy.deepcopy(self._parameters)
        if my_parameter:
            logger.debug("[%s] my_parameter %s", self.log_prefix, my_parameter)
        parent_parameter = self.get_parent_deepcopied_parameters()
        # merge my parameter and parent's
        for k in parent_parameter:
            if k in my_parameter:
                # don't overwrite my parameter
                pass
            else:
                my_parameter[k] = parent_parameter[k]
        if parent_parameter:
            logger.debug("[%s] my_parent_parameter %s", self.log_prefix, parent_parameter)

        if my_parameter:
            logger.debug("[%s] merged_parameter %s", self.log_prefix, my_parameter)
        return my_parameter


class SequentialWorkflowNode(WorkflowNode):
    """Sequential Wrapper Workflow Node"""
    def __init__(self, name, full_path, service_template, node_id, parent_node, executor, flags):
        super().__init__(name, full_path, service_template, node_id, parent_node, executor, flags)
        assert not self.flag_auto_retry, "{} flag_auto_retry cannot be True for s-node".format(self.log_prefix)
        self._in_cleanup_mode = None
        self._passed_fixtures_launch_phase = False

    @property
    def in_cleanup_mode(self):
        return self._in_cleanup_mode

    @in_cleanup_mode.setter
    def in_cleanup_mode(self, value):
        assert value in [AXWorkflowNodeResult.FAILED, AXWorkflowNodeResult.INTERRUPTED]
        if not self._in_cleanup_mode:
            logger.info("%s enter cleanup_mode %s", self.log_prefix, value)
            self._in_cleanup_mode = value

    @property
    def passed_fixtures_launch_phase(self):
        return self._passed_fixtures_launch_phase

    @passed_fixtures_launch_phase.setter
    def passed_fixtures_launch_phase(self, value):
        assert value is True
        assert not self._passed_fixtures_launch_phase
        logger.info("%s enter passed_fixtures_launch_phase", self.log_prefix)
        self._passed_fixtures_launch_phase = value

    def jsonify(self):
        node_key = "{} {}: {}".format(self.get_type_string(), self.node_id, self.state)
        return {node_key: [x.jsonify() for x in self.fixtures_nodes + self.children_nodes]}

    def start(self, is_recover=False, in_cleanup_mode=None):
        logger.info("%s Start node. Recovery mode: %s",
                    self.log_prefix, str(is_recover))
        assert self.is_fresh
        self.state = self.EXPECTING_STATE
        instant_success = False

        if self.dry_run:
            logger.info("%s is dry_run", self.log_prefix)
            instant_success = True
        else:
            if not is_recover or not self.executor.check_event_already_sent(leaf_id=self.node_id,
                                                                            status=ExecutorProducerClient.WAITING_STATE):
                self.report_result_to_kafka(AXWorkflow.get_current_epoch_timestamp_in_ms())

            if self.has_fixtures:
                logger.info("%s starts first fixtures %s",
                            self.log_prefix, self.fixtures_nodes[0])
                cnode = self.fixtures_nodes[0]
                cnode.start(is_recover=is_recover)
            elif len(self.children_nodes) > 0:
                logger.info("%s starts first child %s",
                            self.log_prefix, self.children_nodes[0])
                cnode = self.children_nodes[0]
                cnode.start(is_recover=is_recover)
            else:
                logger.info("%s has instant result", self.log_prefix)
                instant_success = True

        if instant_success:
            result = AXWorkflowNodeResult.factory_for_parent(workflow_id=self.workflow_id,
                                                             node_id=self.node_id,
                                                             result_code=AXWorkflowNodeResult.SUCCEED)
            self.process_result(result=result, is_recover=is_recover)

    def get_type_string(self):
        return 'sequential'

    def _get_max_resource(self):
        result = AXWorkflowResource()

        if self.dry_run:
            return result

        for node in self.children_nodes:
            resource = node.max_resource
            result = AXWorkflowResource.find_max(result, resource)

        for node in self.fixtures_nodes:
            resource = node.max_resource
            result += resource
        return result

    def _get_next_child_to_run(self, index):
        for next_index in range(index + 1, len(self.children_nodes)):
            if self.in_cleanup_mode:
                if not self.children_nodes[next_index].flag_always_run:
                    continue
            return self.children_nodes[next_index]
        return None

    def process_child_result(self, reporting_node, is_recover):
        def generate_and_process_result():
            nonlocal called_process_result_code
            assert called_process_result_code is False
            called_process_result_code = True
            assert self.passed_fixtures_launch_phase or not self.has_fixtures
            result_code = None
            for c_node in self.children_nodes:
                log_prefix = "{} {} is {}".format(self.log_prefix, c_node.node_id, c_node.state)
                if c_node.is_interrupted:
                    assert self.in_cleanup_mode, "{} should be in cleanup mode ".format(log_prefix)
                    result_code = AXWorkflowNodeResult.INTERRUPTED
                    break
                elif c_node.is_failed:
                    if not c_node.flag_ignore_error:
                        assert self.in_cleanup_mode, "{} should be in cleanup mode ".format(log_prefix)
                        result_code = AXWorkflowNodeResult.FAILED
                    else:
                        if not self.in_cleanup_mode:
                            result_code = AXWorkflowNodeResult.SUCCEED
                elif c_node.is_expecting:
                    assert False, "{} bad state".format(log_prefix)
                elif c_node.is_fresh:
                    assert not c_node.flag_always_run, "{} and always run".format(log_prefix)
                    assert self.in_cleanup_mode, "{} should be in cleanup mode ".format(log_prefix)
                elif c_node.is_succeed:
                    if not self.in_cleanup_mode:
                        result_code = AXWorkflowNodeResult.SUCCEED

            if result_code is None:
                assert self.in_cleanup_mode or len(self.children_nodes) == 0
                if self.in_cleanup_mode:
                    result_code = self.in_cleanup_mode
                else:
                    result_code = AXWorkflowNodeResult.SUCCEED

            for c_node in self.fixtures_nodes:
                log_prefix = "{} {} is {}".format(self.log_prefix, c_node.node_id, c_node.state)
                if c_node.is_expecting:
                    assert False, "{} bad state".format(log_prefix)
                elif c_node.is_fresh:
                    assert self.in_cleanup_mode, "{} should be in cleanup mode ".format(log_prefix)

            self.process_result_code(result_code=result_code, is_recover=is_recover)

        def launch_next_or_terminate_fixture(index):
            next_to_launch = self._get_next_child_to_run(index=index)

            if next_to_launch is None:
                # no more child we can launch
                if self.terminate_all_fixtures(is_recover=is_recover):
                    generate_and_process_result()
                else:
                    pass
            else:
                # launch next children
                logger.info("%s launch next child", self.log_prefix)
                if isinstance(next_to_launch, ParallelWorkflowNode):
                    next_to_launch.start(is_recover=is_recover, in_cleanup_mode=self.in_cleanup_mode)
                else:
                    next_to_launch.start(is_recover=is_recover)

        child_result = reporting_node.result
        child_id = reporting_node.node_id
        called_process_result_code = False

        if not reporting_node.is_fixture:
            if self.in_cleanup_mode:
                assert reporting_node.flag_always_run
            assert self.passed_fixtures_launch_phase or not self.has_fixtures or self.in_cleanup_mode
            index = self.children_nodes.index(reporting_node)
            logger.info("%s %s is %s/%s child",
                        self.log_prefix, child_id,
                        index + 1, len(self.children_nodes))

            for i in range(index):
                c_node = self.children_nodes[i]
                if self.in_cleanup_mode:
                    # all previous always_run job must finish
                    if c_node.flag_always_run:
                        assert c_node.is_done
                else:
                    # all previous job must be succeed
                    assert c_node.is_succeed or c_node.flag_ignore_error

            if reporting_node.is_launched:
                # don't need to do anything
                pass
            else:
                assert reporting_node.is_done
                if reporting_node.is_succeed or reporting_node.flag_ignore_error:
                    pass
                else:
                    self.in_cleanup_mode = child_result.result_code

                launch_next_or_terminate_fixture(index=index)
        else:
            # fixture result
            assert reporting_node in self.fixtures_nodes
            logger.info("%s fixture [%s] has result (%s)",
                        self.log_prefix, child_id, child_result.result_code)
            if not self.passed_fixtures_launch_phase:
                assert not self.in_cleanup_mode
                index = self.fixtures_nodes.index(reporting_node)
                logger.info("%s %s is %s/%s fixtures launched",
                            self.log_prefix, child_id, index + 1, len(self.fixtures_nodes))
                for i in range(index):
                    # all previous fixture must be launched or done
                    nd = self.fixtures_nodes[i]
                    assert nd.is_launched or nd.is_done

                with self._fixtures_termination_triggered_lock:
                    termination_triggered = self._fixtures_termination_triggered

                if reporting_node.is_failed or reporting_node.is_interrupted or termination_triggered:
                    self.in_cleanup_mode = child_result.result_code
                    self.passed_fixtures_launch_phase = True

                    launch_next_or_terminate_fixture(index=-1)
                else:
                    assert reporting_node.is_launched or reporting_node.is_succeed
                    if reporting_node.is_launched and (isinstance(reporting_node, ParallelWorkflowNode) and not reporting_node.all_children_launched):
                        logger.info("%s ignore parallel %s launched result", self.log_prefix, reporting_node.node_id)
                    else:
                        if len(self.fixtures_nodes) - 1 == index:
                            self.passed_fixtures_launch_phase = True
                            if len(self.children_nodes) > 0:
                                # start first step
                                logger.info("%s launch first child", self.log_prefix)
                                self.children_nodes[0].start(is_recover=is_recover)
                            else:
                                # no step
                                if self.terminate_all_fixtures(is_recover=is_recover):
                                    # all fixtures have been terminated
                                    generate_and_process_result()
                                else:
                                    pass
                        else:
                            # launch next fixture if hasn't
                            next_node = self.fixtures_nodes[index + 1]
                            if next_node.is_fresh:
                                logger.info("%s start %sth fixture",
                                            self.log_prefix, index + 2)
                                next_node.start(is_recover=is_recover)
                            else:
                                logger.info("%s %sth fixture already started",
                                            self.log_prefix, index + 2)
            else:
                # no longer need to launch fixture (either all fixture are all launched or in_cleanup_mode)
                pending_fixture = {}
                for cnode in self.fixtures_nodes:
                    if not self.in_cleanup_mode:
                        assert not cnode.is_fresh
                        assert not cnode.is_expecting
                    if cnode.is_expecting_or_launched:
                        pending_fixture[cnode.node_id] = cnode.name

                pending_children = {}
                for cnode in self.children_nodes:
                    if cnode.is_expecting_or_launched:
                        pending_children[cnode.node_id] = cnode.name

                if (not pending_fixture) and (not pending_children):
                    generate_and_process_result()
                else:
                    logger.info("%s still waiting pending-fix=%s pending-child=%s",
                                self.log_prefix, pending_fixture, pending_children)

        if self.is_expecting and not called_process_result_code:
            self.process_result_code(result_code=AXWorkflowNodeResult.LAUNCHED, is_recover=is_recover)


class ParallelWorkflowNode(WorkflowNode):
    """Parallel Wrapper Workflow Node"""
    def __init__(self, name, full_path, service_template, node_id, parent_node, executor, is_fixture):
        super().__init__(name, full_path, service_template, node_id, parent_node, executor, flags={})
        self.is_fixture = is_fixture
        self.all_children_launched = False
        self._in_cleanup_mode = None

        assert not self.flag_always_run, "{} flag_always_run cannot be True for p-node".format(self.log_prefix)
        assert not self.flag_auto_retry, "{} flag_auto_retry cannot be True for p-node".format(self.log_prefix)
        assert not self.flag_ignore_error, "{} flag_ignore_error cannot be True for p-node".format(self.log_prefix)

    @property
    def in_cleanup_mode(self):
        return self._in_cleanup_mode

    def jsonify(self):
        node_key = "{} {}: {}".format(self.get_type_string(), self.node_id, self.state)
        node_val = {}
        for x in self.fixtures_nodes + self.children_nodes:
            node_val[x.node_id] = x.jsonify()
        return {node_key: node_val}

    def start(self, is_recover=False, in_cleanup_mode=None):
        logger.info('Start node [%s]. Recovery mode: %s', repr(self), str(is_recover))
        assert self.is_fresh
        self.state = self.EXPECTING_STATE

        instant_success = False

        if self.dry_run:
            logger.info("%s is dry_run", self.log_prefix)
            instant_success = True
        else:
            if in_cleanup_mode:
                assert self.flag_always_run

            logger.info("%s starts all children (p) %s",
                        self.log_prefix, self.get_fixtures_ids() + self.get_children_ids())

            self._in_cleanup_mode = in_cleanup_mode
            cnode_start = 0
            if len(self.fixtures_nodes + self.children_nodes) > 0:
                for cnode in self.fixtures_nodes + self.children_nodes:
                    if cnode.flag_always_run or not in_cleanup_mode:
                        cnode.start(is_recover=is_recover)
                        cnode_start += 1

            if cnode_start == 0:
                assert not in_cleanup_mode
                assert not self.flag_always_run
                logger.info("%s starts no child (in_cleanup_mode=%s) and has instant result",
                            self.log_prefix, in_cleanup_mode)
                instant_success = True

        if instant_success:
            result = AXWorkflowNodeResult.factory_for_parent(workflow_id=self.workflow_id,
                                                             node_id=self.node_id,
                                                             result_code=AXWorkflowNodeResult.SUCCEED)
            self.process_result(result=result, is_recover=is_recover)

    def get_type_string(self):
        return 'parallel'

    def _get_max_resource(self):
        result = AXWorkflowResource()

        if self.dry_run:
            return result

        if self.is_fixture:
            for node in self.fixtures_nodes:
                resource = node.max_resource
                result += resource
        else:
            for node in self.children_nodes:
                resource = node.max_resource
                result = AXWorkflowResource.find_max(result, resource)
        return result

    def process_child_result(self, reporting_node, is_recover):
        logger.info("%s has %s child",
                    self.log_prefix, len(self.children_nodes))

        failed = []
        interrupted = []
        expecting = []
        succeed = []
        launched = []
        skipped = []

        for cnode in self.fixtures_nodes + self.children_nodes:
            cnode_id = cnode.node_id
            if cnode.is_failed:
                if cnode.flag_ignore_error:
                    succeed.append(cnode_id)
                else:
                    failed.append(cnode_id)
            elif cnode.is_interrupted:
                interrupted.append(cnode_id)
            elif cnode.is_fresh or cnode.is_expecting:
                if cnode.flag_always_run or not self._in_cleanup_mode:
                    expecting.append(cnode_id)
                    # still have children not done
                    if cnode.is_fresh:
                        logger.warning("%s child %s still fresh",
                                       self.log_prefix, cnode.node_id)
                else:
                    skipped.append(cnode_id)
            elif cnode.is_launched:
                launched.append(cnode_id)
            elif cnode.is_succeed:
                succeed.append(cnode_id)
            else:
                assert False

        logger.info("%s failed=%s interrupted=%s expecting=%s succeed=%s launched=%s skipped=%s",
                    self.log_prefix, failed, interrupted, expecting, succeed, launched, skipped)

        report_launched_to_parent = False
        if self.all_children_launched:
            assert not expecting
        else:
            if not expecting:
                logger.info("%s all_children_launched", self.log_prefix)
                self.all_children_launched = True
                if self.is_launched:
                    report_launched_to_parent = True

        if self.is_fixture:
            parent_node = self.parent_node
            assert parent_node.has_fixtures
            if not parent_node.passed_fixtures_launch_phase:
                if expecting:
                    # still wait child to start
                    result_code = None
                elif failed:
                    # need to terminate the rest
                    s = expecting + launched
                    if s:
                        for c in s:
                            cnode = self.executor.get_node(c)
                            assert cnode.is_leaf and cnode.is_fixture
                            assert not cnode.terminate_all_fixtures(is_recover=is_recover)
                        result_code = None
                    else:
                        result_code = AXWorkflowNodeResult.FAILED
                elif interrupted:
                    if launched:
                        # terminate all launched
                        for c in launched:
                            cnode = self.executor.get_node(c)
                            assert cnode.is_leaf and cnode.is_fixture
                            assert not cnode.terminate_all_fixtures(is_recover=is_recover)
                        result_code = None
                    else:
                        result_code = AXWorkflowNodeResult.INTERRUPTED
                elif launched:
                    result_code = AXWorkflowNodeResult.LAUNCHED
                else:
                    result_code = AXWorkflowNodeResult.SUCCEED
            else:
                if expecting:
                    assert self.in_cleanup_mode or parent_node.in_cleanup_mode
                    result_code = None
                elif launched:
                    result_code = None
                elif failed:
                    result_code = AXWorkflowNodeResult.FAILED
                elif interrupted:
                    result_code = AXWorkflowNodeResult.INTERRUPTED
                else:
                    result_code = AXWorkflowNodeResult.SUCCEED
        else:
            if expecting:
                result_code = None
            elif launched:
                result_code = None
            elif failed:
                result_code = AXWorkflowNodeResult.FAILED
            elif interrupted:
                result_code = AXWorkflowNodeResult.INTERRUPTED
            else:
                result_code = AXWorkflowNodeResult.SUCCEED

        # this node is done or some children already have problems
        if result_code is not None:
            self.process_result_code(result_code=result_code, is_recover=is_recover)
        else:
            logger.info("%s still expecting child", self.log_prefix)
            if self.is_expecting or report_launched_to_parent:
                self.process_result_code(result_code=AXWorkflowNodeResult.LAUNCHED, is_recover=is_recover)

        return


class LeafWorkflowNode(WorkflowNode):
    """Leaf Workflow Node (Abstract)"""
    def __init__(self, name, full_path, service_template, node_id, parent_node, executor, flags):
        super().__init__(name, full_path, service_template, node_id, parent_node, executor, flags)
        self.resource = None
        self._is_deployment = False
        self._is_dind = False
        self.first_time_see_image_pull = None
        try:
            if service_template["template"]["labels"]["ax_ea_deployment"]:
                self._is_deployment = True
                logger.info("%s is deployment", self.log_prefix)
        except Exception:
            pass

        try:
            ax_ea_docker_enable_payload = service_template["template"]["labels"]["ax_ea_docker_enable"]
            if ax_ea_docker_enable_payload:
                logger.info("%s is dind", self.log_prefix)
                self._is_dind = True
        except Exception:
            pass

    @property
    def is_deployment(self):
        return self._is_deployment

    @property
    def is_dind(self):
        return self._is_dind

    @property
    def is_leaf(self):
        return True

    def jsonify(self):
        node_key = "{} {}: {}".format(self.get_type_string(), self.node_id, self.state)
        return {node_key: repr(self.result)}

    def get_type_string(self):
        raise NotImplementedError

    def _get_max_resource(self):
        raise NotImplementedError

    def start(self, is_recover=False, in_cleanup_mode=None):
        logger.info("%s Start node. Recovery mode: %s",
                    self.log_prefix, str(is_recover))
        assert self.is_fresh
        self.state = self.EXPECTING_STATE

        instant_success = False

        if self.dry_run:
            logger.info("%s is dry_run", self.log_prefix)
            instant_success = True
        else:
            if not is_recover or not self.executor.check_event_already_sent(leaf_id=self.node_id,
                                                                            status=ExecutorProducerClient.WAITING_STATE):
                self.report_result_to_kafka(AXWorkflow.get_current_epoch_timestamp_in_ms())
            if not is_recover:  # Start the container thread
                self.executor.start_start_and_monitor_container_thread(self, False)

        if instant_success:
            result = AXWorkflowNodeResult.factory_for_parent(workflow_id=self.workflow_id,
                                                             node_id=self.node_id,
                                                             result_code=AXWorkflowNodeResult.SUCCEED)
            self.process_result(result=result, is_recover=is_recover)


class UserContainerNode(LeafWorkflowNode):
    """User Container Node"""
    def __init__(self, name, full_path, service_template, node_id, parent_node, executor, resource, flags):
        super().__init__(name, full_path, service_template, node_id, parent_node, executor, flags)
        self.resource = resource

    def get_type_string(self):
        return 'user-container'

    def _get_max_resource(self):
        if self.dry_run:
            return AXWorkflowResource()
        if self.is_done:
            return AXWorkflowResource()

        result = self.resource + AXWorkflowResource(WorkflowNode.ARTIFACT_RESOURCE)
        return result

    def terminate_all_fixtures(self, is_recover):
        raise NotImplementedError


class FixtureWorkflowNode(LeafWorkflowNode):
    """Static Fixture Node"""
    def __init__(self, name, full_path, service_template, node_id, parent_node, executor, flags):
        super().__init__(name, full_path, service_template, node_id, parent_node, executor, flags)
        self.is_fixture = True
        assert not self.flag_always_run, "{} flag_always_run cannot be True for fixture".format(self.log_prefix)

    def get_type_string(self):
        raise NotImplementedError

    def _get_max_resource(self):
        raise NotImplementedError

    def terminate_all_fixtures(self, is_recover):
        with self._fixtures_termination_triggered_lock:
            if self.is_leaf and self._fixtures_termination_triggered:
                logger.info("%s termination already triggered", self.log_prefix)
                return False
            self._fixtures_termination_triggered = True

        logger.info("%s start termination thread is_recover=%s", self.log_prefix, is_recover)
        self._fixtures_need_to_be_terminated = True
        if not is_recover:
            self.start_terminate_fixture_container_thread()
        return False

    def _terminate_fixture_container_thread(self):
        try:
            self._terminate_fixture_container()
        except Exception as e:
            logger.exception("got exception")
            sleep_second = 20
            logger.info("%s exception. sleep %s seconds", self.log_prefix, sleep_second)
            time.sleep(sleep_second)
            self.executor.record_last_exception_event(exception=e)
            logger.error("%s os._exit(3). will restart", self.log_prefix)
            os._exit(3)

    def start_terminate_fixture_container_thread(self):
        logger.info("%s start termination thread", self.log_prefix)
        t = threading.Thread(name=self.get_type_string() + "-terminator-" + self.node_id,
                             target=self._terminate_fixture_container_thread, args=())
        t.daemon = True
        t.start()

    def _terminate_fixture_container(self):
        fixture_termination_list_key = AXWorkflow.REDIS_FIXTURE_TERMINATION_LIST_KEY.format(self.node_id)
        v = {"killed_by": self.node_id}
        logger.info("%s about to send termination signal to %s",
                    self.log_prefix, fixture_termination_list_key)
        redis_client.rpush(key=fixture_termination_list_key, value=v,
                           expire=AXWorkflow.REDIS_LIST_EXPIRE_SECONDS, encoder=json.dumps)
        logger.info("%s termination signal sent", self.log_prefix)


class DynamicFixtureNode(FixtureWorkflowNode):
    """Dynamic Fixture Node"""
    def __init__(self, name, full_path, service_template, node_id, parent_node, executor, resource, flags):
        super().__init__(name, full_path, service_template, node_id, parent_node, executor, flags)
        self.resource = resource

    def get_type_string(self):
        return 'dynamic'

    def _get_max_resource(self):
        if self.dry_run:
            return AXWorkflowResource()
        if self.is_done:
            return AXWorkflowResource()

        result = self.resource + AXWorkflowResource(WorkflowNode.ARTIFACT_RESOURCE)
        return result


class StaticFixtureNode(FixtureWorkflowNode):
    """Static Fixture Node"""
    def __init__(self, name, full_path, service_template, node_id, parent_node, executor, flags):
        super().__init__(name, full_path, service_template, node_id, parent_node, executor, flags)
        self._volume_reserve_result = dict()

        # Separate requirements and volume requirements
        tmp_requirements = self.service_template.get('requirements', {})
        self._vol_requirements = tmp_requirements.get('merged_volume_mounts', {})

        # Remove volume requirements from requirements
        tmp_requirements.pop('merged_volume_mounts', None)
        self._requirements = tmp_requirements

    def get_type_string(self):
        return 'static'

    def _get_max_resource(self):
        return AXWorkflowResource()

    @property
    def static_fixture_requirements(self):
        return self._requirements

    @property
    def static_volume_requirements(self):
        return self._vol_requirements

    @property
    def get_volume_reserve_result(self):
        return self._volume_reserve_result

    def set_volume_reserve_result(self, result_json):
        self._volume_reserve_result = result_json


class DeploymentNode(LeafWorkflowNode):
    """Deploymeent Node"""
    def __init__(self, name, full_path, service_template, node_id, parent_node, executor, flags):
        super().__init__(name, full_path, service_template, node_id, parent_node, executor, flags)
        self.wait_deployment_up = flags.get("wait_deployment_up", True)

    def get_type_string(self):
        return 'deployment'

    def _get_max_resource(self):
        return AXWorkflowResource()


class AXWorkflowExecutor(object):
    def __init__(self, workflow_id, self_container_name, report_done_url, ax_sys_cpu_core=0.225, ax_sys_mem_mib=560, vol_size=100.0, instance_type='m3.large', fake_run=False):
        super(AXWorkflowExecutor, self).__init__()
        self._workflow_id = workflow_id
        self._self_container_name = self_container_name
        self._report_done_url = report_done_url
        self._service_template = None
        self._root_node = None
        self._workflow = None
        self._node_events = None

        self._is_test_workflow = False
        self._test_crash_second = 0
        self._test_expected_failure_node = None

        self._nodes = {}

        self._current_sn = -1
        self._results_q = []
        self._results_q_cond = threading.Condition()
        self._shutdown = False

        self._send_heartbeat = False
        self._can_send_nodes_status = False
        self._total_session = 0

        self._last_exception_recoded_lock = threading.Lock()
        self._last_exception_recoded = False

        self._fake_run = fake_run  # For d3 drawing purpose

        self.log_prefix = AXWorkflowExecutor.get_log_prefix(self._workflow_id)

        self._ax_sys_cpu_core = float(ax_sys_cpu_core)
        self._ax_sys_mem_mib = float(ax_sys_mem_mib)
        self._vol_size = float(vol_size)
        self._instance_type = str(instance_type)

    @staticmethod
    def get_log_prefix(workflow_id):
        return "[WFE] [{}]:".format(workflow_id)

    def init(self):
        self._workflow = AXWorkflow.get_workflow_by_id_from_db(workflow_id=self.workflow_id, need_load_template=True)
        try:
            assert self._workflow is not None
            self._service_template = self._workflow.service_template
            assert self._service_template
            assert self._service_template.get("id") == self.workflow_id
        except Exception:
            logger.exception("%s: invalid workflow_id", self.log_prefix)
            self.stop_self_container()
            assert False
            # should not reach here

        if AXWorkflow.tag_test_ax_workflow in self._service_template:
            test_tags = self._service_template.pop(AXWorkflow.tag_test_ax_workflow)
            self._is_test_workflow = True
            self._test_crash_second = test_tags.get(AXWorkflow.tag_test_ax_workflow_executor_crash_second, 0)
            self._test_expected_failure_node = test_tags.get(AXWorkflow.tag_test_ax_workflow_expect_failure_leaf_node, 0)

        container_status = axsys_client.get_container_status(container_name=self._self_container_name)

        # Update global table for artifact tags
        artifact_tags = self._service_template.get("artifact_tags", None)
        if artifact_tags:
            try:
                tag_list = json.loads(artifact_tags)
                if tag_list and isinstance(tag_list, list):
                    logger.info("%s has artifact tags: %s", self.log_prefix, str(tag_list))
                    metadata = axdb_client.get_artifact_meta('artifact_tags')
                    prev_artifact_tags_str = metadata['value']
                    prev_artifact_tag_list = json.loads(prev_artifact_tags_str)
                    for tag in tag_list:
                        if tag not in prev_artifact_tag_list:
                            prev_artifact_tag_list.append(tag)
                    prev_artifact_tag_list = sorted(prev_artifact_tag_list)
                    next_artifact_tags_str = json.dumps(prev_artifact_tag_list)
                    if next_artifact_tags_str != prev_artifact_tags_str:
                        axdb_client.update_artifact_meta_conditionally('artifact_tags', next_artifact_tags_str, prev_artifact_tags_str)
                    logger.info('Successfully updated global metadata (attribute: artifact_tags)')
            except Exception:
                logger.exception("Failed to update global metadata (attribute: artifact_tags), %s", artifact_tags)

        logger.info("%s container %s in %s", self.log_prefix, self._self_container_name, container_status)

    @property
    def workflow_id(self):
        return self._workflow_id

    @property
    def is_test_workflow(self):
        return self._is_test_workflow

    @staticmethod
    def startup_prerequisite(workflow_id):
        # Basic logging.
        logger.info("%s starting...", AXWorkflowExecutor.get_log_prefix(workflow_id))
        logger.info("%s AXDB version: %s",
                    AXWorkflowExecutor.get_log_prefix(workflow_id), AXWorkflow.get_db_version_wait_till_db_is_ready())
        redis_client.wait(timeout=30 * 60)
        logger.info("%s redis is available", AXWorkflowExecutor.get_log_prefix(workflow_id))

    def _increase_current_sn(self):
        # assert self._results_q_cond.locked() xxx python condition doesn't have locked?
        self._current_sn += 1
        return self._current_sn

    def _recover(self):
        results = self._get_workflow_results_from_db()

        # start the root node
        if len(results):
            self._node_events = self._get_workflow_node_events_from_db()
            logger.info("%s recover(%s steps) start.",
                        self.log_prefix, len(results))
            self._root_node.start(is_recover=True)

            # replay all results
            for result in results:
                node = self.get_node(result.node_id)

                if result.sn != self._current_sn + 1:
                    logger.critical("%s serial number from db result does not match!! sn_from_db=%s, current_sn=%s, sn_from_db != current_sn + 1",
                                    self.log_prefix, result.sn, self._current_sn)
                    assert result.sn == self._current_sn + 1

                node.process_result(result=result, is_recover=True)
                self._current_sn = result.sn

            # free up the space for node events
            self._node_events = None
            logger.info("%s recover done.", self.log_prefix)
        else:
            logger.info("%s no need for recover", self.log_prefix)

        expecting_after_recover = {}
        launched_after_recover = {}
        expecting_after_recover_non_leaf = {}
        launched_after_recover_non_leaf = {}
        # make sure all proper containers are up
        for node_id, node in self._nodes.items():
            if node.is_leaf:
                if node.is_expecting_or_launched:
                    # we need to launch container
                    if node.is_expecting:
                        expecting_after_recover[node_id] = node.name
                    else:
                        launched_after_recover[node_id] = node.name
                    if not self._fake_run:
                        self.start_start_and_monitor_container_thread(node, part_of_recover=True)
            else:
                if node.is_expecting:
                    expecting_after_recover_non_leaf[node_id] = node.name
                elif node.is_launched:
                    launched_after_recover_non_leaf[node_id] = node.name

        logger.info("%s %s+%s leaf nodes %s %s in expecting/launched state after recover.",
                    self.log_prefix, len(expecting_after_recover), len(launched_after_recover),
                    expecting_after_recover, launched_after_recover)
        logger.info("%s %s+%s non-leaf nodes %s %s in expecting/launched state after recover.",
                    self.log_prefix, len(expecting_after_recover_non_leaf), len(launched_after_recover_non_leaf),
                    expecting_after_recover_non_leaf, launched_after_recover_non_leaf)
        logger.info("%s after recover %s", self.log_prefix, self._get_nodes_stats())

        return

    def _wait_for_state_to_be_running_or_running_del_or_done(self):
        logger.info("%s wait adc change workflow status", self.log_prefix)
        while True:
            workflow = AXWorkflow.get_workflow_by_id_from_db(self._workflow_id)
            assert workflow
            assert workflow.status not in [AXWorkflow.SUSPENDED], "{} bad workflow status".format(self.log_prefix)
            if workflow.status in [AXWorkflow.RUNNING_DEL, AXWorkflow.RUNNING_DEL_FORCE, AXWorkflow.RUNNING,
                                   AXWorkflow.DELETED, AXWorkflow.SUCCEED, AXWorkflow.FAILED, AXWorkflow.FORCED_FAILED]:
                return

            logger.info("%s status=%s, still waiting ...", self.log_prefix, workflow.status)
            time.sleep(10)

    def _start_if_have_not(self):
        # make sure all proper containers are up
        root_is_fresh = False
        has_other_state = False
        for node_id, node in self._nodes.items():
            if node.is_root and node.is_fresh:
                root_is_fresh = True
            if not node.is_fresh:
                has_other_state = True
        if root_is_fresh:
            assert not has_other_state
            self._wait_for_state_to_be_running_or_running_del_or_done()
            logger.info("%s kickstart", self.log_prefix)
            self._root_node.start(is_recover=False)
        else:
            logger.info("%s no need for kickstart", self.log_prefix)
        logger.info("%s Workflow Tree after start \n%s", self.log_prefix, str(self._root_node))

    @staticmethod
    def _construct_full_path(prefix, name):
        if name:
            if prefix:
                return "{}.{}".format(prefix, name)
            else:
                return name
        else:
            return prefix

    @staticmethod
    def build_tree(workflow_id, payload, name, full_path, parent_node, is_fixture, leaf_nodes, executor, resource_payload, dry_run=False):
        """
        Recursively build up the workflow tree.

        The function is specifically tailored to the current structure of the service template.
        For future changes on service template, this build_tree might not be suitable.
        """
        def check_duplidate_id(node_id, node):
            if node_id in leaf_nodes:
                logger.error("%s Two leaf nodes (1)[%s], (2)[%s] have same id", log_prefix, repr(leaf_nodes[node_id]),
                             repr(node))
                raise AXIllegalArgumentException("{} Two leaf nodes have same id {}".format(log_prefix, node_id))

        try:
            node_id = payload.get('id', None)
            flags = payload.get('flags', {})
            template = payload.get('template', None)
            log_prefix = AXWorkflowExecutor.get_log_prefix(workflow_id)
            if template:
                type = template.get('type', None)
            else:
                type = 'static fixture'
            if type == 'static fixture':  # Static fixture
                assert is_fixture, "{} service template name [{}] is {}".format(log_prefix, name, payload)
                assert node_id is None, "Expecting no id set for static fixtures"
                node_id = str(uuid.uuid1())
                if name == 'merged_static_fixture':
                    st = {'id': node_id, 'requirements': payload}
                else:
                    st = {'id': node_id, 'requirements': {name: payload}}
                node = StaticFixtureNode(name=name,
                                         full_path=full_path,
                                         service_template=st,
                                         node_id=node_id, parent_node=parent_node,
                                         executor=executor, flags=flags)  # Static fixture node
                check_duplidate_id(node_id, node)
            elif type == 'deployment':
                assert not is_fixture,  "{} service template name [{}] is {}".format(log_prefix, name, payload)
                node = DeploymentNode(name=name,
                                      full_path=full_path,
                                      service_template=payload,
                                      node_id=node_id, parent_node=parent_node,
                                      executor=executor, flags=flags)  # Deployment node
                check_duplidate_id(node_id, node)
            elif 'volumes' not in template and 'steps' not in template and 'fixtures' not in template: # workflow
                assert type == 'container'
                resource = AXWorkflowExecutor.process_leaf_node_resource_helper(leaf_template=template, ax_cpu_core=resource_payload['ax_cpu_core'],
                                                                                ax_mem_mib=resource_payload['ax_mem_mib'], max_vol_size=resource_payload['max_vol_size'],
                                                                                instance_type=resource_payload['instance_type'], dry_run=dry_run)
                if is_fixture:
                    node = DynamicFixtureNode(name=name, full_path=full_path, service_template=payload,
                                              node_id=node_id, parent_node=parent_node,
                                              executor=executor, resource=resource, flags=flags)  # Dynamic fixture node
                else:
                    node = UserContainerNode(name=name, full_path=full_path, service_template=payload,
                                             node_id=node_id, parent_node=parent_node,
                                             executor=executor, resource=resource, flags=flags)  # User container node
                check_duplidate_id(node_id, node)
            else:
                assert type == 'workflow'
                node = SequentialWorkflowNode(name=name, full_path=full_path, service_template=template,
                                              node_id=node_id, parent_node=parent_node,
                                              executor=executor, flags=flags)  # Sequential wrapper node
                # parse volumes section and put the part into fixtures section:
                vols = template.get('volumes', {})

                if vols and dry_run is False:
                    # The idea is to convert volumes section into static fixture
                    agg_vol_payload = dict()
                    for key, value in vols.items():
                        if 'name' in value:
                            # This is a named volume
                            agg_vol_payload[key] = {
                                'axrn': 'vol:/{}'.format(value.get('name', ""))
                            }
                        elif 'storage_class' in value:
                            agg_vol_payload[key] = {
                                'storage_class': value.get('storage_class', ""),
                                'size_gb': value.get('size_gb', "")
                            }
                        else:
                            logger.error("Failed to parse the volume %s payload: %s", key, value)
                            continue

                    if 'fixtures' not in template:
                        template['fixtures'] = list()

                    logger.info("template before volume change %s", json.dumps(template, indent=2))
                    template['fixtures'].append({'merged_volume_mounts': agg_vol_payload})
                    logger.info("template after volume change %s", json.dumps(template, indent=2))

                for key_word in ['fixtures', 'steps']:
                    is_fixtures_steps = (key_word == "fixtures")
                    attribute_name = 'fixtures_nodes' if is_fixtures_steps else 'children_nodes'
                    steps = template.get(key_word, [])
                    for idx, step in enumerate(steps):
                        assert isinstance(step, dict), "{} Bad template {} with {} not being a dictionary".format(log_prefix, template, key_word)
                        if len(step) == 0:
                            continue
                        elif len(step) == 1:
                            for key, value in step.items():
                                child_node = AXWorkflowExecutor.build_tree(workflow_id, value, key,
                                                                           AXWorkflowExecutor._construct_full_path(full_path, key),
                                                                           node, is_fixtures_steps, leaf_nodes, executor, resource_payload, dry_run)
                                getattr(node, attribute_name).append(child_node)
                        else:
                            p_node_str = str('{}.{}.{}'.format(node_id, idx, key_word).encode('utf-8'))
                            p_node_id = str(uuid.uuid5(uuid.NAMESPACE_X500, p_node_str))
                            p_node_name = '{}_{}_{}'.format(name, ('p_f' if is_fixtures_steps else 'p'), idx)
                            p_node = ParallelWorkflowNode(name=p_node_name, service_template=step,
                                                          full_path=AXWorkflowExecutor._construct_full_path(full_path, p_node_name),
                                                          node_id=p_node_id, parent_node=node,
                                                          executor=executor, is_fixture=is_fixtures_steps)  # Parallel wrapper node
                            static_fixture_requirements = {}
                            for key, value in step.items():
                                if is_fixtures_steps and 'template' not in value:  # Static fixture
                                    assert key not in static_fixture_requirements, "{} Duplicate static fixtures {} in parallel step.".format(log_prefix, steps)
                                    static_fixture_requirements[key] = value
                                else:  # Dynamic fixture
                                    p_node_child = AXWorkflowExecutor.build_tree(workflow_id, value, key,
                                                                                 AXWorkflowExecutor._construct_full_path(full_path, key),
                                                                                 p_node, is_fixtures_steps, leaf_nodes, executor, resource_payload, dry_run)
                                    getattr(p_node, attribute_name).append(p_node_child)
                            if static_fixture_requirements:  # Merge parallel static fixtures into single template
                                p_node_child = AXWorkflowExecutor.build_tree(workflow_id, static_fixture_requirements,
                                                                             'merged_static_fixture',
                                                                             full_path,
                                                                             p_node, is_fixtures_steps, leaf_nodes, executor, resource_payload, dry_run)
                                getattr(p_node, attribute_name).append(p_node_child)
                            getattr(node, attribute_name).append(p_node)
            leaf_nodes[node_id] = node
        except AXIllegalArgumentException as e:
            raise e
        except Exception as e:
            raise AXIllegalArgumentException(str(e))

        return node

    @staticmethod
    def process_leaf_node_resource_helper(leaf_template, ax_cpu_core, ax_mem_mib, max_vol_size, instance_type, is_dind=False, dry_run=False):
        cpu_core = float(leaf_template['resources']['cpu_cores'])
        mem_mib = float(leaf_template['resources']['mem_mib'])
        instance_cpu_core = INSTANCE_RESOURCE[instance_type][0]
        instance_mem_mib = INSTANCE_RESOURCE[instance_type][1]
        sidecar_cpu_core = 0 if is_dind else WorkflowNode.ARTIFACT_RESOURCE[0]
        sidecar_mem_mib = 0 if is_dind else WorkflowNode.ARTIFACT_RESOURCE[1]
        new_cpu_core, _ = AXWorkflowExecutor.process_leaf_node_resource(cpu_core=cpu_core,
                                                                        mem_mib=mem_mib,
                                                                        disk_gb=0,
                                                                        instance_cpu_core=instance_cpu_core,
                                                                        instance_mem_mib=instance_mem_mib,
                                                                        instance_disk_gb=max_vol_size,
                                                                        ax_cpu_core=ax_cpu_core,
                                                                        ax_mem_mib=ax_mem_mib,
                                                                        sidecar_cpu_core=sidecar_cpu_core,
                                                                        sidecar_mem_mib=sidecar_mem_mib)
        if not dry_run:
            leaf_template['resources']['cpu_cores'] = new_cpu_core

        # Handle the dind case
        try:
            ax_ea_docker_enable_payload = json.loads(leaf_template["labels"]["ax_ea_docker_enable"])
            dind_cpu_core = float(ax_ea_docker_enable_payload['cpu_cores'])
            dind_mem_mib = float(ax_ea_docker_enable_payload['mem_mib'])
            dind_disk_gb = int(ax_ea_docker_enable_payload['graph-storage-size'].split('Gi')[0])
            if ax_ea_docker_enable_payload:
                dind_new_cpu_core, _ = AXWorkflowExecutor.process_leaf_node_resource(cpu_core=dind_cpu_core,
                                                                                     mem_mib=dind_mem_mib,
                                                                                     disk_gb=dind_disk_gb,
                                                                                     instance_cpu_core=instance_cpu_core,
                                                                                     instance_mem_mib=instance_mem_mib,
                                                                                     instance_disk_gb=max_vol_size,
                                                                                     ax_cpu_core=ax_cpu_core,
                                                                                     ax_mem_mib=ax_mem_mib,
                                                                                     sidecar_cpu_core=sidecar_cpu_core,
                                                                                     sidecar_mem_mib=sidecar_mem_mib)
                # Update the info in the payload
                if not dry_run:
                    ax_ea_docker_enable_payload['cpu_cores'] = dind_new_cpu_core
                    leaf_template["labels"]["ax_ea_docker_enable"] = json.dumps(ax_ea_docker_enable_payload)

                logger.info("Old ax_ea_docker_enable label: %s, new ax_ea_docker_enable label: %s",
                            leaf_template["labels"]["ax_ea_docker_enable"], json.dumps(ax_ea_docker_enable_payload))

                # Adding up to the main container
                logger.info("Main container resource: cpu: %s, memory: %s, dind contaienr resource: cpu: %s, memory %s",
                            new_cpu_core, mem_mib, dind_new_cpu_core, dind_mem_mib)
                new_cpu_core += dind_new_cpu_core
                mem_mib += dind_mem_mib

        except Exception:
            pass
        return AXWorkflowResource([new_cpu_core, mem_mib])

    @staticmethod
    def process_leaf_node_resource(cpu_core, mem_mib, disk_gb, instance_cpu_core, instance_mem_mib, instance_disk_gb, ax_cpu_core, ax_mem_mib, sidecar_cpu_core, sidecar_mem_mib):
        """
        This function converts the resource specified by the user to the resource that is supposed to submit to Axmon.
        Note the calculation is highly customized to the current design, and will be improved for future changes.

        :param cpu_core: user requested cpu cores
        :param mem_mib: user requested memory in megabytes
        :param disk_gb: user requested disk space in gigabytes
        :param instance_cpu_core: cpu per instance
        :param instance_mem_mib: memory per instance
        :param instance_disk_gb: volume size per instance
        :param ax_cpu_core: cpu cores used by AX per instance
        :param ax_mem_mib: memory used by AX per instance
        :param sidecar_cpu_core: cpu cores used by artifact container
        :param sidecar_mem_mib: memory used by artifact container
        :return new_cpu_core:
        :return new_mem_mib:
        """
        # At most we scale down to 70 percent of what it was before
        logger.info("Twisting resource: \n cpu_core %s\n mem_mib %s \n disk_gb %s \n instance_cpu_core %s \n instance_mem_mib %s \n instance_disk_gb %s \n ax_cpu_core %s \n ax_mem_mib %s \n sidecar_cpu_core %s \n sidecar_mem_mib %s \n",
                    cpu_core, mem_mib, disk_gb, instance_cpu_core, instance_mem_mib, instance_disk_gb, ax_cpu_core, ax_mem_mib, sidecar_cpu_core, sidecar_mem_mib)

        minimum_resource_scale = MINIMUM_RESOURCE_SCALE

        cpu_core = float(cpu_core)
        mem_mib = float(mem_mib)
        disk_gb = float(disk_gb)

        cpu_core_ratio = cpu_core / instance_cpu_core
        mem_mib_ratio = mem_mib / instance_mem_mib
        disk_gb_ratio = disk_gb / instance_disk_gb

        logger.info("cpu ratio: %s", cpu_core_ratio)

        # If disk ratio is bigger than cpu ratio, we bump up cpu ratio to be same as disk ratio
        if disk_gb_ratio > cpu_core_ratio:
            cpu_core_ratio = disk_gb_ratio

        # New resource = (node capacity - ax usage) * ratio - sidecar resource
        new_cpu_core = float(instance_cpu_core - ax_cpu_core) * cpu_core_ratio - sidecar_cpu_core
        new_mem_mib = float(instance_mem_mib - ax_mem_mib) * mem_mib_ratio - sidecar_mem_mib

        # Check whether the newly calculated resource is too small
        if new_cpu_core / cpu_core < minimum_resource_scale:
            new_cpu_core = cpu_core * minimum_resource_scale

            if new_cpu_core < 0.001:  # Avoid too small of a number
                new_cpu_core = 0.001

        # For now, do not scale memory
        # if new_mem_mib / mem_mib < minimum_resource_scale:
        #     new_mem_mib = math.floor(mem_mib * minimum_resource_scale)  # Round down to avoid float problem

        logger.info("Convert resources cpu %s and memory %s to %s and %s", cpu_core, mem_mib, new_cpu_core, mem_mib)
        return new_cpu_core, mem_mib

    @staticmethod
    def set_ignore_delete_interrupt(node):
        node.ignore_delete_interrupt = node.flag_always_run or (node.parent_node is not None and node.parent_node.ignore_delete_interrupt)
        for n in node.children_nodes + node.fixtures_nodes:
            AXWorkflowExecutor.set_ignore_delete_interrupt(n)

        if isinstance(node, ParallelWorkflowNode):
            for n in node.children_nodes:
                if n.flag_always_run:
                    node.flag_always_run = True

    def _build_nodes(self):
        logger.info("%s build_nodes from %s", self.log_prefix,
                    {} if self._total_session > 1 else pprint.pformat(self._service_template))
        resource_payload = {
            'ax_cpu_core': self._ax_sys_cpu_core,
            'ax_mem_mib': self._ax_sys_mem_mib,
            'max_vol_size': self._vol_size,
            'instance_type': self._instance_type,
        }
        self._root_node = AXWorkflowExecutor.build_tree(workflow_id=self.workflow_id, payload=self._service_template, name='root', full_path='',
                                                        parent_node=None, is_fixture=False, leaf_nodes=self._nodes, executor=self, resource_payload=resource_payload)
        AXWorkflowExecutor.set_ignore_delete_interrupt(self._root_node)

        logger.info("%s build_nodes %s", self.log_prefix, self._get_nodes_stats())
        logger.info("%s Workflow Tree after build \n%s", self.log_prefix, str(self._root_node))

    def _get_nodes_stats(self):
        stats = {
            "type": {
                "total": 0,
                "leaf": 0,
                "leaf_fixture": 0,
                "parallel": 0,
                "serial": 0},
            "state": {
                "succeed": 0,
                "failed": 0,
                "expecting": 0,
                "launched": 0,
                "interrupted": 0,
                "fresh": 0,
            },
            "state_leaf": {
                "succeed": 0,
                "failed": 0,
                "expecting": 0,
                "launched": 0,
                "interrupted": 0,
                "fresh": 0,
                "failed_nodes": {}
            },
            "state_leaf_fixture": {
                "succeed": 0,
                "failed": 0,
                "expecting": 0,
                "launched": 0,
                "interrupted": 0,
                "fresh": 0,
                "failed_nodes": {}
            }
        }
        stats_type = stats["type"]
        stats_state = stats["state"]
        stats_type["total"] = len(self._nodes)
        for key, value in self._nodes.items():
            if value.is_leaf:
                if value.is_fixture:
                    stats_type["leaf_fixture"] += 1
                    stats_leaf = stats["state_leaf_fixture"]
                else:
                    stats_type["leaf"] += 1
                    stats_leaf = stats["state_leaf"]
                if value.is_fresh:
                    stats_leaf["fresh"] += 1
                if value.is_expecting:
                    stats_leaf["expecting"] += 1
                if value.is_launched:
                    stats_leaf["launched"] += 1
                if value.is_succeed:
                    stats_leaf["succeed"] += 1
                if value.is_failed:
                    stats_leaf["failed"] += 1
                    stats_leaf["failed_nodes"][value.node_id] = {"name": value.get_service_template_name()}
                    if value.result:
                        stats_leaf["failed_nodes"][value.node_id]["result"] = value.result.jsonify()
                if value.is_interrupted:
                    stats_leaf["interrupted"] += 1
            if isinstance(value, ParallelWorkflowNode):
                stats_type["parallel"] += 1
            if isinstance(value, SequentialWorkflowNode):
                stats_type["serial"] += 1
            if value.is_fresh:
                stats_state["fresh"] += 1
            if value.is_expecting:
                stats_state["expecting"] += 1
            if value.is_launched:
                stats_state["launched"] += 1
            if value.is_succeed:
                stats_state["succeed"] += 1
            if value.is_failed:
                stats_state["failed"] += 1
            if value.is_interrupted:
                stats_state["interrupted"] += 1

        return stats

    def _wait_and_process_results(self):
        while not self._shutdown:
            with self._results_q_cond:
                logger.debug("%s [controller] Q wait", self.log_prefix)
                if len(self._results_q) == 0 and not self._shutdown:
                    self._results_q_cond.wait()
                logger.debug("%s [controller] Q wakeup", self.log_prefix)
                results = []
                while len(self._results_q) and not self._shutdown:
                    result = self._results_q.pop(0)
                    results.append(result)
                logger.info("%s [controller] dequeue %s results", self.log_prefix, len(results))
            for r in results:
                node = self.get_node(r.node_id)
                node.process_result(result=r, is_recover=False)
                # logger.info("%s Workflow Tree after process result \n%s", self.log_prefix, str(self._root_node))
        logger.info("%s [controller] leave wait_and_process_results", self.log_prefix)

    def _add_result_to_q(self, node_id, name, result_code, detail):
        with self._results_q_cond:
            sn = self._increase_current_sn()
            result = AXWorkflowNodeResult(workflow_id=self._workflow_id,
                                          node_id=node_id, sn=sn,
                                          detail=detail, result_code=result_code,
                                          timestamp=AXWorkflow.get_current_epoch_timestamp_in_ms())
            logger.info("%s [worker] [%s] [%s] add result (sn=%s result_code=%s detail=%s) to q",
                        self.log_prefix, node_id, name, sn, result_code, detail)
            self._results_q.append(result)
            self._results_q_cond.notifyAll()

    def _start_and_monitor_container(self, node, root_service_template):
        if isinstance(node, StaticFixtureNode):
            return self._start_and_monitor_static_fixture(node=node,
                                                          root_service_template=root_service_template)
        elif isinstance(node, DeploymentNode):
            return self._start_and_wait_deployment_start(node=node,
                                                         root_service_template=root_service_template)
        else:
            return self._start_and_monitor_dynamic_fixture_or_normal_container(node=node,
                                                                               root_service_template=root_service_template)

    def _start_and_monitor_static_fixture(self, node, root_service_template):
        def retry_exception_func(exception):
            """
            Retry based on exception raised from GET method
            :param exception:
            :return:
            """
            logger.error("%s exception=%s", node.log_prefix, exception)
            if isinstance(exception, (AttributeError, TypeError, KeyError)):
                return False
            if isinstance(exception, requests.HTTPError):
                try:
                    if 500 <= exception.response.status_code < 600:
                        # retry if code is 5xx
                        return True
                    else:
                        return False
                except Exception:
                    # retry if no code
                    return True

            return True

        @retry(wait_exponential_multiplier=1000,
               wait_exponential_max=60000,
               stop_max_attempt_number=60,
               retry_on_exception=retry_exception_func)
        def request_reserve_fixture():
            requirements = copy.deepcopy(node.static_fixture_requirements)
            vol_requirements = copy.deepcopy(node.static_volume_requirements)
            user_info = root_service_template.get('user', 'admin@internal')
            logger.info("Fixture requirements, %s. Volume requirements %s, user %s", requirements, vol_requirements, user_info)
            mock_count = 0
            for req_name, req in requirements.items():
                if req and "is_ax_test_mock" in req:
                    mock_count += 1
                    req.pop("is_ax_test_mock")
            # have to be all or none mock
            if mock_count != 0 and mock_count != len(requirements):
                logger.error("%s have to be all or none mock %s", node.log_prefix, node.static_fixture_requirements)
                raise TypeError
            url = "http://fixturemanager.axsys:8912/v1/fixture/requests{}".format('_mock' if mock_count > 0 else "")
            nonlocal json_to_fixturemanager
            json_to_fixturemanager = {"service_id": node.node_id,
                                      "root_workflow_id": node.workflow_id,
                                      "requirements": requirements,
                                      "vol_requirements": vol_requirements,
                                      "requester": "axworkflowadc",
                                      "user": user_info,}
            logger.info("%s post %s to %s (original %s)",
                        node.log_prefix, json_to_fixturemanager, url, node.static_fixture_requirements)
            response = requests.post(url, json=json_to_fixturemanager)
            response.raise_for_status()
            return response.json()

        @retry(wait_exponential_multiplier=1000,
               wait_exponential_max=60000,
               stop_max_attempt_number=60,
               retry_on_exception=retry_exception_func)
        def release_reserve_fixture():
            url = "http://fixturemanager.axsys:8912/v1/fixture/requests/{}".format(node.node_id)
            logger.info("%s release fix fixture by delete %s", node.log_prefix, url)
            response = requests.delete(url)
            response.raise_for_status()

            return response.json()

        def process_reservation_result(reservation_json, vol_reservation_json):
            try:
                result = {}
                vol_result = {}
                bad_result = []
                requirements = node.static_fixture_requirements
                vol_requirements = node.static_volume_requirements

                for f in reservation_json:
                    if isinstance(reservation_json[f], dict):
                        result[f] = reservation_json[f]
                    else:
                        result[f] = {}
                    if f not in requirements:
                        bad_result.append(f)

                for f in vol_reservation_json:
                    if isinstance(vol_reservation_json[f], dict):
                        result[f] = vol_reservation_json[f]
                        vol_result[f] = {"details": vol_reservation_json[f]}
                    else:
                        result[f] = {}
                        vol_result = {}
                    if f not in vol_requirements:
                        bad_result.append(f)

                if bad_result or len(reservation_json) != len(requirements) or len(vol_reservation_json) != len(vol_requirements):
                    logger.error("bad reservation return %s vs %s, vol: %s vs %s. bad_result=%s",
                                 reservation_json, requirements, vol_reservation_json, vol_requirements, bad_result)
                    return AXWorkflowNodeResult.FAILED, reservation_json
                else:
                    # Set volume result into the node
                    node.set_volume_reserve_result(vol_result)
                    return AXWorkflowNodeResult.LAUNCHED, result
            except Exception:
                logger.exception("%s", node.log_prefix)

            return AXWorkflowNodeResult.FAILED, None

        def check_reservation_available():
            try:
                response = requests.get("http://fixturemanager.axsys:8912/v1/fixture/requests/{}".format(node.node_id))
                if response.status_code == 404:
                    return None, None
                response.raise_for_status()
                fix_req = response.json()
                if fix_req['assignment'] or fix_req['vol_assignment']:
                    logger.info("%s got reservation signal %s", node.log_prefix, fix_req)
                    return process_reservation_result(fix_req['assignment'], fix_req['vol_assignment'])
            except Exception:
                logger.exception("%s", node.log_prefix)
            return None, None

        def check_terminate_signal():
            if node.fixtures_need_to_be_terminated:
                logger.info("%s noticed fixtures_need_to_be_terminated", node.log_prefix)
                return AXWorkflowNodeResult.SUCCEED, None

            workflow = AXWorkflow.get_workflow_by_id_from_db(workflow_id=node.workflow_id)
            if workflow.status in [AXWorkflow.RUNNING_DEL, AXWorkflow.RUNNING_DEL_FORCE, AXWorkflow.DELETED]:
                if workflow.status == AXWorkflow.RUNNING_DEL:
                    if node.ignore_delete_interrupt:
                        logger.info("%s always_run ignore %s state", node.log_prefix, workflow.status)
                        return None, None
                    elif node.is_fixture and not check_reserve_available:
                        logger.info("%s fixture launched, ignore %s state", node.log_prefix, workflow.status)
                        return None, None
                if workflow.status == AXWorkflow.DELETED:
                    logger.warning("%s interrupted, but state is already %s. This should not happen",
                                   node.log_prefix, AXWorkflow.DELETED)
                else:
                    logger.info("%s in %s state", node.log_prefix, workflow.status)
                return AXWorkflowNodeResult.INTERRUPTED, None

            return None, None

        def wait_reserve_fixture(check_reservation=False):
            wait_seconds = 60 * 5
            keys = [fixture_termination_list_key, workflow_del_force_list_key]
            if node.ignore_delete_interrupt:
                pass
            elif node.is_fixture and not check_reservation:
                pass
            else:
                keys.append(workflow_del_list_key)
            if check_reservation:
                keys.append(fixture_available_list_key)
            logger.info("%s wait for %s", node.log_prefix, keys)
            tuple_val = redis_client.brpop(keys, timeout=wait_seconds)

            if tuple_val is not None:
                if not (isinstance(tuple_val, tuple) and len(tuple_val) >= 2):
                    logger.info("%s ignore bad redis return %s", node.log_prefix, tuple_val)
                    return None, None

                if tuple_val[0] == fixture_available_list_key:
                    reservation_result = tuple_val[1]
                    logger.info("%s wakeup result=%s from %s",
                                node.log_prefix, reservation_result, tuple_val[0])
                    return check_reservation_available()
                elif tuple_val[0] in [workflow_del_list_key, workflow_del_force_list_key]:
                    logger.info("%s wakeup interrupted by %s", node.log_prefix, tuple_val[0])
                    # requeue message so other threads can pick it up too
                    redis_client.rpush(key=tuple_val[0], value=tuple_val[1],
                                       expire=AXWorkflow.REDIS_LIST_EXPIRE_SECONDS)
                    return AXWorkflowNodeResult.INTERRUPTED, None
                elif tuple_val[0] == fixture_termination_list_key:
                    logger.info("%s wakeup by %s", node.log_prefix, tuple_val[0])
                    return AXWorkflowNodeResult.SUCCEED, None
                else:
                    assert False, "{} bad redis return {}".format(node.log_prefix, tuple_val)
            else:
                return None, None

        fixture_termination_list_key = AXWorkflow.REDIS_FIXTURE_TERMINATION_LIST_KEY.format(node.node_id)
        fixture_available_key = AXWorkflow.REDIS_FIXTURE_ASSIGNMENT_KEY.format(node.node_id)
        fixture_available_list_key = AXWorkflow.REDIS_FIXTURE_ASSIGNMENT_LIST_KEY.format(node.node_id)
        workflow_del_list_key = AXWorkflow.REDIS_DEL_LIST_KEY.format(node.workflow_id)
        workflow_del_force_list_key = AXWorkflow.REDIS_DEL_FORCE_LIST_KEY.format(node.workflow_id)

        json_to_fixturemanager = {}
        need_to_reserve = check_reserve_available = node.is_expecting

        while True:
            result_code, detail = check_terminate_signal()
            if result_code is not None:
                assert result_code in [AXWorkflowNodeResult.SUCCEED, AXWorkflowNodeResult.INTERRUPTED]
                release_reserve_fixture()
                return result_code, detail

            if check_reserve_available:
                result_code, detail = check_reservation_available()
                if result_code is not None:
                    if result_code == AXWorkflowNodeResult.LAUNCHED:
                        check_reserve_available = False
                        need_to_reserve = False
                        detail = {
                            "to_fixture_manager": json_to_fixturemanager,
                            "launch_type": "fixture",
                            "output_parameters": detail
                        }
                        self._add_result_to_q(node_id=node.node_id, name=node.name,
                                              result_code=result_code,
                                              detail=detail)
                        logger.info('%s result (%s) enqueued',
                                    node.log_prefix, result_code)
                    else:
                        logger.info('%s bad result %s %s',
                                    node.log_prefix, result_code, detail)
                        return AXWorkflowNodeResult.FAILED, {"error": detail}

            if need_to_reserve:
                try:
                    request_reserve_fixture()
                    need_to_reserve = False
                except Exception as e:
                    logger.exception("%s", node.log_prefix)
                    if isinstance(e, requests.HTTPError):
                        try:
                            detail = e.response.json()
                            error_code = detail.get('code', '')
                            logger.error("%s error=%s, response=%s", node.log_prefix, error_code, detail)
                        except Exception:
                            logger.exception("%s", node.log_prefix)
                            detail = str(e)
                        return AXWorkflowNodeResult.FAILED, {"error": detail}
                    else:
                        raise

            # just monitor notification
            wait_reserve_fixture(check_reservation=check_reserve_available)

    def _start_and_wait_deployment_start(self, node, root_service_template):
        def retry_exception_func(exception):
            """
            Retry based on exception raised from GET method
            :param exception:
            :return:
            """
            logger.error("%s exception=%s", node.log_prefix, exception)
            if isinstance(exception, (AttributeError, TypeError, KeyError)):
                return False
            if isinstance(exception, requests.HTTPError):
                try:
                    if 500 <= exception.response.status_code < 600:
                        # retry if code is 5xx
                        return True
                    else:
                        return False
                except Exception:
                    # retry if no code
                    return True

            return True

        @retry(wait_exponential_multiplier=1000,
               wait_exponential_max=60000,
               stop_max_attempt_number=60,
               retry_on_exception=retry_exception_func)
        def request_deployment():
            result, _ = check_terminate_signal()
            if result is not None:
                assert result in [AXWorkflowNodeResult.INTERRUPTED]
                return result

            # xxx todo: we need a better way to handle deployment fixture
            service_template = service_template_pre_process(root_service_template,
                                                            leaf=node.service_template,
                                                            parameter=node.get_parent_deepcopied_parameters(),
                                                            name=node.name,
                                                            full_path=node.full_path)
            spec = service_template

            url = "http://axamm.axsys:8966/v1/deployments"  # xxx amm todo
            nonlocal json_to_amm
            # json_to_amm = {"service_id": node.node_id, "root_workflow_id": node.workflow_id, "spec": spec}
            json_to_amm = spec
            del json_to_amm['status']
            logger.info("%s post %s to %s", node.log_prefix, json_to_amm, url)
            response = requests.post(url, json=json_to_amm)
            response.raise_for_status()
            return response.json()

        def process_redis_deployment_result(deployment_result):
            try:
                deployment_json_result = json.loads(deployment_result)
                deployment_result_status = deployment_json_result.get("status")
                deployment_result_detail = deployment_json_result.get("status_detail")
                if deployment_result_status in ["Error", "Terminated", "Stopped"]:
                    logger.error("bad deployment_result return %s", deployment_json_result)
                    return AXWorkflowNodeResult.FAILED, deployment_result_detail
                elif deployment_result_status in ["Active"]:  # xxx amm, Waiting needs to be removed later
                    return AXWorkflowNodeResult.SUCCEED, deployment_result_detail
                elif deployment_result_status in ["Waiting"]:
                    self._add_result_to_q(node_id=node.node_id, name=node.name,
                                          result_code=AXWorkflowNodeResult.LAUNCHED,
                                          detail=deployment_result_detail)
                    return None, None
                elif deployment_result_status in ["Terminating", "Stopping"]:
                    # For Terminating and Stopping, we do not want to stop the workflow just yet.
                    # Instead, we report as running state to kafka to let frontend know it is in
                    # running state.
                    status_payload = dict()
                    status_payload['service_id'] = node.node_id
                    status_payload['status'] = ExecutorProducerClient.RUNNING_STATE
                    status_payload['status_detail'] = deployment_result_detail
                    self.do_report_to_kafka(node.node_id, status_payload)
                    return None, None
                else:
                    return None, None
            except Exception:
                logger.exception("%s: got deployment_result %s", node.log_prefix, deployment_result)
                return AXWorkflowNodeResult.FAILED, None

        def check_deployment_is_up():
            result = redis_client.get(deployment_is_up_key)
            if result:
                logger.info("%s got deployment signal %s", node.log_prefix, result)
                return process_redis_deployment_result(result)
            return None, None

        def check_terminate_signal():
            workflow = AXWorkflow.get_workflow_by_id_from_db(workflow_id=node.workflow_id)
            if workflow.status in [AXWorkflow.RUNNING_DEL, AXWorkflow.RUNNING_DEL_FORCE, AXWorkflow.DELETED]:
                if workflow.status == AXWorkflow.DELETED:
                    logger.warning("%s interrupted, but state is already %s. This should not happen",
                                   node.log_prefix, AXWorkflow.DELETED)
                else:
                    logger.info("%s in %s state", node.log_prefix, workflow.status)
                return AXWorkflowNodeResult.INTERRUPTED, None

            return None, None

        def wait_deployment_up():
            wait_seconds = 60 * 5
            keys = [workflow_del_force_list_key, workflow_del_list_key]
            keys.append(deployment_is_up_list_key)
            logger.info("%s wait for %s", node.log_prefix, keys)
            tuple_val = redis_client.brpop(keys, timeout=wait_seconds)

            if tuple_val is not None:
                if not (isinstance(tuple_val, tuple) and len(tuple_val) >= 2):
                    logger.info("%s ignore bad redis return %s", node.log_prefix, tuple_val)
                    return

                if tuple_val[0] == deployment_is_up_list_key:
                    br_result = tuple_val[1]
                    logger.info("%s wakeup result=%s from %s",
                                node.log_prefix, br_result, tuple_val[0])
                    return
                elif tuple_val[0] in [workflow_del_list_key, workflow_del_force_list_key]:
                    logger.info("%s wakeup interrupted by %s", node.log_prefix, tuple_val[0])
                    # requeue message so other threads can pick it up too
                    redis_client.rpush(key=tuple_val[0], value=tuple_val[1],
                                       expire=AXWorkflow.REDIS_LIST_EXPIRE_SECONDS)
                    return
                else:
                    assert False, "{} bad redis return {}".format(node.log_prefix, tuple_val)
            else:
                return

        deployment_is_up_key = AXWorkflow.REDIS_DEPLOYMENT_UP_KEY.format(node.node_id)
        deployment_is_up_list_key = AXWorkflow.REDIS_DEPLOYMENT_UP_LIST_KEY.format(node.node_id)
        workflow_del_list_key = AXWorkflow.REDIS_DEL_LIST_KEY.format(node.workflow_id)
        workflow_del_force_list_key = AXWorkflow.REDIS_DEL_FORCE_LIST_KEY.format(node.workflow_id)

        json_to_amm = {}
        need_to_request_deployment = True
        while True:
            result_code, detail = check_terminate_signal()
            if result_code is not None:
                assert result_code in [AXWorkflowNodeResult.INTERRUPTED]
                return result_code, detail

            result_code, detail = check_deployment_is_up()
            if result_code is not None:
                if result_code == AXWorkflowNodeResult.SUCCEED:
                    return result_code, detail
                elif result_code == AXWorkflowNodeResult.LAUNCHED:
                    return result_code, detail
                else:
                    logger.info('%s bad result %s %s', node.log_prefix, result_code, detail)
                    return AXWorkflowNodeResult.FAILED, detail

            if need_to_request_deployment:
                try:
                    result_deployment = request_deployment()
                    if result_deployment == AXWorkflowNodeResult.INTERRUPTED:
                        return AXWorkflowNodeResult.INTERRUPTED, None

                    need_to_request_deployment = False
                except Exception as e:
                    logger.exception("%s", node.log_prefix)
                    if isinstance(e, requests.HTTPError):
                        try:
                            detail = e.response.json()
                            error_code = detail.get('code', AXWorkflowNodeResult.FAILED_CANNOT_CONNECT_APPLICATION_MANAGER)
                            error_msg = detail.get('message', str(e))
                            logger.error("%s error=%s, response=%s", node.log_prefix, error_code, detail)
                            return AXWorkflowNodeResult.FAILED, {"code": error_code, "message": error_msg}
                        except Exception:
                            logger.exception("%s", node.log_prefix)
                        return AXWorkflowNodeResult.FAILED, {"code": AXWorkflowNodeResult.FAILED_CANNOT_CONNECT_APPLICATION_MANAGER, "message": str(e)}
                    else:
                        raise

            if node.wait_deployment_up:
                wait_deployment_up()
            else:
                logger.info("%s no wait for deployment", node.log_prefix)
                return AXWorkflowNodeResult.SUCCEED, None


    def _substitute_params(self, template, leaf_node):
        """
        TODO: This function is here as the class StaticFixtureNode is defined in this file.
        """
        if "arguments" not in template:
            return
        if not leaf_node.parent_node:
            # We can get here if the job is just a single container job
            return

        # create a map for volumes
        vol_map = dict()
        for fixture_node in leaf_node.parent_node.fixtures_nodes or []:
            if not isinstance(fixture_node, StaticFixtureNode):
                continue
            vol_map.update(fixture_node.get_volume_reserve_result.items())

        # First we substitute any %%variable%% in the arguments array
        # i.e. the arguments itself need to be expanded first. This is especially true for volumes (fixtures?)
        for arg_key, arg_val in template['arguments'].items():
            logger.debug("Processing argument {} {}".format(arg_key, arg_val))
            match = re.match(r'^%%volumes\.([-a-zA-Z0-9_]+)%%?', arg_val)
            if not match:
                continue
            vol_name = match.groups()[0]

            details = vol_map.get(vol_name, None)
            if details is None:
                logger.debug("Did not find volume details for {} in {}".format(vol_name, json.dumps(vol_map)))
                continue

            logger.debug("Subsitute value of argument {} from {} to {}".format(arg_key, arg_val, details))
            template['arguments'][arg_key] = details

        # Now that arguments have been expanded, subsitute container params with input arguments
        if "inputs" not in template["template"] or "volumes" not in template["template"]["inputs"]:
            logger.debug("No input volumes in template => No parameter substitution")
            return

        for k, v in template["template"]["inputs"]["volumes"].items():
            logger.debug("Processing input volume {} {}".format(k, json.dumps(v)))
            arg_val = template["arguments"].get("volumes.{}".format(k), None)
            if arg_val is not None:
                logger.debug("Updating input volume {} {} with {}".format(k, json.dumps(v), json.dumps(arg_val)))
                merged_val = v
                merged_val.update(arg_val)
                template["template"]["inputs"]["volumes"][k] = merged_val


    def _start_and_monitor_dynamic_fixture_or_normal_container(self, node, root_service_template):
        def process_redis_task_result(result):
            try:
                jr = json.loads(result)
                event_type = jr.get("event_type", None)
                if event_type == "HAVE_RESULT":
                    detail = {AXWorkflowNodeResult.DETAIL_TAG_CONTAINER_RETURN_JSON: jr}
                    rr = jr.get("return_code", default_bad_return_code)
                    err_msg = jr.get("message", None)
                    k8s_info_container_status = jr.get("k8s_info", {}).get("container_status", {})
                    if rr == 0:  # Should be compared with expected rc
                        retn = AXWorkflowNodeResult.SUCCEED, detail
                    elif rr == 137 and termination_deletion_issued > 0:
                        if node.is_fixture:
                            detail[AXWorkflowNodeResult.DETAIL_TAG_FIXTURE_TERMINATED_BY_EXECUTOR] = {}
                        elif node.is_deployment:
                            detail[AXWorkflowNodeResult.DETAIL_TAG_DEPLOYMENT_TERMINATED_BY_REQUEST] = {}
                        else:
                            assert False
                        retn = AXWorkflowNodeResult.SUCCEED, detail
                    elif rr == 10001:
                        detail[AXWorkflowNodeResult.DETAIL_TAG_FAILURE_REASON] = AXWorkflowNodeResult.FAILED_CANNOT_FIND_RETURN
                        retn = AXWorkflowNodeResult.FAILED, detail
                    elif rr == 10002:
                        detail[AXWorkflowNodeResult.DETAIL_TAG_FAILURE_REASON] = AXWorkflowNodeResult.FAILED_CANNOT_LOAD_ARTIFACT
                        detail[AXWorkflowNodeResult.DETAIL_TAG_FAILURE_MESSAGE] = err_msg
                        retn = AXWorkflowNodeResult.FAILED, detail
                    elif rr == 10003:
                        detail[AXWorkflowNodeResult.DETAIL_TAG_FAILURE_REASON] = AXWorkflowNodeResult.FAILED_NOT_ALLOW_RETRY
                        retn = AXWorkflowNodeResult.FAILED, detail
                    elif rr == 10005:
                        detail[AXWorkflowNodeResult.DETAIL_TAG_FAILURE_REASON] = AXWorkflowNodeResult.FAILED_CANNOT_SAVE_ARTIFACT
                        detail[AXWorkflowNodeResult.DETAIL_TAG_FAILURE_MESSAGE] = err_msg
                        retn = AXWorkflowNodeResult.FAILED, detail
                    elif rr == 146:
                        detail[AXWorkflowNodeResult.DETAIL_TAG_FAILURE_REASON] = AXWorkflowNodeResult.FAILED_SIG_TERM
                        retn = AXWorkflowNodeResult.FAILED, detail
                    else:
                        oom = False
                        for container_name in k8s_info_container_status:
                            try:
                                if k8s_info_container_status[container_name]["reason"] == "OOMKilled":
                                    logger.info("%s, %s is OOMKilled", log_prefix, container_name)
                                    oom = True
                            except Exception:
                                pass

                        detail[AXWorkflowNodeResult.DETAIL_TAG_FAILURE_REASON] = AXWorkflowNodeResult.FAILED_NONE_ZERO_RETURN if not oom else AXWorkflowNodeResult.FAILED_OOM_KILLED
                        retn = AXWorkflowNodeResult.FAILED, detail
                elif event_type == "LOADING_ARTIFACTS":
                    if not node.load_artifacts_timestamp:
                        node.load_artifacts_timestamp = jr.get("timestamp", 1)
                        if node.is_expecting:
                            node.report_result_to_kafka()
                    return None, None
                elif event_type == "SAVING_ARTIFACTS":
                    if not node.save_artifacts_timestamp:
                        node.save_artifacts_timestamp = jr.get("timestamp", 1)
                        if node.is_launched:
                            node.report_result_to_kafka()
                    return None, None
                else:
                    assert False, "bad event {} in {}".format(event_type, jr)

            except Exception:
                logger.exception("%s bad redis return %s", log_prefix, result)
                retn = AXWorkflowNodeResult.FAILED, {AXWorkflowNodeResult.DETAIL_TAG_CONTAINER_RETURN_BAD: result}

            return retn

        def container_launch_ack(cont_launch_result):
            nonlocal local_is_expecting
            assert local_is_expecting
            local_is_expecting = False
            logger.info("%s, get launch callback %s", log_prefix, cont_launch_result)
            # acked it unconditionally for now
            axdb_client.put_workflow_kv(key=launch_ack_key, value={"allow": 1})
            redis_client.rpush(key=launch_ack_list_key, value={},
                               expire=AXWorkflow.REDIS_LIST_EXPIRE_SECONDS, encoder=json.dumps)
            report_container_is_launched(cont_launch_result)

        def report_container_is_launched(cont_launch_result):
            try:
                cont_launch_result = json.loads(cont_launch_result)
            except Exception:
                logger.exception("%s bad result %s", log_prefix, cont_launch_result)
                cont_launch_result = {}
            detail = {
                "service_template": service_template,
                "launch_type": "dynamic" if node.is_fixture else "normal",
                "output_parameters": {
                    node.name: cont_launch_result
                }
            }
            self._add_result_to_q(node_id=node.node_id, name=node.name,
                                  result_code=AXWorkflowNodeResult.LAUNCHED,
                                  detail=detail)
            logger.info('%s result (%s) enqueued',
                        log_prefix, AXWorkflowNodeResult.LAUNCHED)

        def wait_container_status_change():
            wait_seconds = 60 * 5
            wait_list = [task_result_list_key, workflow_del_force_list_key]
            if local_is_expecting:
                wait_list.append(launch_list_key)
            if termination_deletion_issued <= 0 and node.is_fixture:
                wait_list.append(fixture_termination_list_key)
            if node.ignore_delete_interrupt:
                pass
            elif node.is_fixture and not local_is_expecting:
                pass
            elif node.is_deployment and termination_deletion_issued > 0:
                pass
            else:
                wait_list.append(workflow_del_list_key)

            logger.debug("%s wait on redis keys %s", log_prefix, wait_list)
            tuple_val = redis_client.brpop(wait_list, timeout=wait_seconds)

            # test missing event
            if global_test_mode:
                if random.randint(0, 10) < 5:
                    logger.debug("%s test_mode drop redis event", log_prefix)
                    return False

            if tuple_val is not None:
                if not (isinstance(tuple_val, tuple) and len(tuple_val) >= 2):
                    logger.info("%s ignore bad redis return %s", log_prefix, tuple_val)
                    return False

                if tuple_val[0] == task_result_list_key:
                    container_result = tuple_val[1]
                    logger.info("[%s wakeup %s result=%s",
                                log_prefix, tuple_val[0], container_result)
                    return True
                elif tuple_val[0] in [workflow_del_list_key, workflow_del_force_list_key]:
                    logger.info("%s wakeup interrupted by %s",
                                log_prefix, tuple_val[0])
                    # requeue message so other threads can pick it up too
                    redis_client.rpush(key=tuple_val[0], value=tuple_val[1],
                                       expire=AXWorkflow.REDIS_LIST_EXPIRE_SECONDS)
                    return True
                elif tuple_val[0] == launch_list_key:
                    logger.info("%s wakeup %s", log_prefix, tuple_val[0])
                    return True
                elif tuple_val[0] == fixture_termination_list_key:
                    logger.info("%s wakeup %s", log_prefix, tuple_val[0])
                    return True
                else:
                    assert False, "{} bad redis return {}".format(log_prefix, tuple_val)
            else:
                logger.info("%s waited %s seconds", log_prefix, wait_seconds)
                return False

        # start of the function
        default_bad_return_code = 20001
        local_is_expecting = node.is_expecting
        container_has_started = False
        missing_container_retry_left = 10
        termination_deletion_issued = 0

        log_prefix = node.log_prefix

        task_result_key = AXWorkflow.WFL_RESULT_KEY.format(node.node_id)              # Key of the task result in Redis
        task_result_list_key = AXWorkflow.REDIS_RESULT_LIST_KEY.format(node.node_id)    # Key of the task result list for BRPOP
        workflow_del_list_key = AXWorkflow.REDIS_DEL_LIST_KEY.format(node.workflow_id)  # Key of the workflow DELETE list for BRPOP
        workflow_del_force_list_key = AXWorkflow.REDIS_DEL_FORCE_LIST_KEY.format(node.workflow_id)
        fixture_termination_list_key = AXWorkflow.REDIS_FIXTURE_TERMINATION_LIST_KEY.format(node.node_id)
        launch_key = AXWorkflow.WFL_LAUNCH_KEY.format(node.node_id)
        launch_list_key = AXWorkflow.REDIS_LAUNCH_LIST_KEY.format(node.node_id)
        launch_ack_key = AXWorkflow.WFL_LAUNCH_ACK_KEY.format(node.node_id)
        launch_ack_list_key = AXWorkflow.REDIS_LAUNCH_ACK_LIST_KEY.format(node.node_id)

        # pre_process
        try:
            service_template = service_template_pre_process(root_service_template,
                                                            leaf=node.service_template,
                                                            parameter=node.get_parent_deepcopied_parameters(),
                                                            name=node.name,
                                                            full_path=node.full_path)

            logger.debug("ST after preprocess {}".format(json.dumps(service_template)))
            self._substitute_params(service_template, node)
            logger.debug("ST after arg subs {}".format(json.dumps(service_template)))

            auto_retry = node.flag_auto_retry
            if auto_retry is None:
                auto_retry = not node.is_fixture
            AXWorkflow.service_template_add_reporting_callback_param(service_template=service_template,
                                                                     instance_id=node.node_id,
                                                                     auto_retry=auto_retry,
                                                                     is_wfe=False)
            logger.info("%s dry run to get ucname", log_prefix)
            r, cont, response_status_code, response_json = axsys_client.create_service(service_template, dry_run=True)
        except (requests.ConnectionError, requests.Timeout) as e:
            logger.exception("%s create_service dry timeout=%s", log_prefix)
            self.record_last_exception_event(exception=e)
            logger.error("%s os._exit(11). will restart", log_prefix)
            os._exit(11)
        except Exception:
            logger.exception("%s bad service_template=%s",
                             log_prefix, node.service_template)
            return AXWorkflowNodeResult.FAILED, {
                AXWorkflowNodeResult.DETAIL_TAG_FAILURE_REASON: AXWorkflowNodeResult.FAILED_BAD_TEMPLATE}

        if r:
            uc_name = axsys_client.canonical_container_full_name(cont[0])  # User container name
            uc_name_guessed = AxsysClient.guess_container_full_name(service_template)
            if uc_name != uc_name_guessed:
                logger.critical("%s name not matching real=%s vs guessed=%s. node=%s %s",
                                log_prefix, uc_name, uc_name_guessed, node, service_template)
            node.service_name = uc_name
            log_prefix = "[WFE] [{}] [{}]:".format(node.workflow_id, uc_name)

        else:
            logger.error("%s cannot do dry run. node=%s %s. response_status_code=%s response_json=%s",
                         log_prefix, node, service_template, response_status_code, response_json)
            return AXWorkflowNodeResult.FAILED, {
                AXWorkflowNodeResult.DETAIL_TAG_FAILURE_REASON: AXWorkflowNodeResult.FAILED_CANNOT_LAUNCH_CONTAINER_DRY,
                "response_status_code": response_status_code,
                AXWorkflowNodeResult.DETAIL_TAG_FAILURE_MESSAGE: json.dumps(response_json)
            }

        node.first_time_see_image_pull = None
        got_result = False
        while True:
            workflow = AXWorkflow.get_workflow_by_id_from_db(workflow_id=node.workflow_id)
            assert workflow, "{} no workflow".format(log_prefix)
            if workflow.status in [AXWorkflow.SUSPENDED, AXWorkflow.ADMITTED_DEL,
                                   AXWorkflow.ADMITTED, AXWorkflow.SUCCEED,
                                   AXWorkflow.FAILED, AXWorkflow.FORCED_FAILED]:
                # this should never happen
                container_status = axsys_client.get_container_status(container_name=uc_name)
                if container_status in [AxsysClient.CONTAINER_RUNNING, axsys_client.CONTAINER_IMAGE_PULL_BACKOFF,
                                        AxsysClient.CONTAINER_PENDING, AxsysClient.CONTAINER_STOPPED,
                                        AxsysClient.CONTAINER_FAILED]:
                    logger.info("%s is still in %s. delete", log_prefix, container_status)
                    rc_del, result_del = axsys_client.delete_service(service_name=uc_name, force=True)
                    logger.info("%s delete return %s %s",
                                log_prefix, rc_del, result_del)
                assert False, "{} bad status {} {}, this should never happen".format(log_prefix, workflow.status, container_status)
            elif workflow.status in [AXWorkflow.RUNNING, AXWorkflow.RUNNING_DEL]:
                if workflow.status in [AXWorkflow.RUNNING_DEL]:
                    if node.ignore_delete_interrupt:
                        logger.info("%s always_run, ignore %s", log_prefix, workflow.status)
                    elif node.is_fixture and not local_is_expecting:
                        logger.info("%s fixture is launched, ignore %s", log_prefix, workflow.status)
                    elif node.is_deployment:
                        if local_is_expecting:
                            # force deleting container, need no callback
                            rc_del, result_del = axsys_client.delete_service(service_name=uc_name, force=True)
                            logger.info("%s force delete return %s %s", log_prefix, rc_del, result_del)
                            ret = AXWorkflowNodeResult.SUCCEED, {
                                AXWorkflowNodeResult.DETAIL_TAG_DEPLOYMENT_TERMINATED_BEFORE_LAUNCH: True}
                            break
                        else:
                            rc_del, result_del = axsys_client.delete_service(service_name=uc_name, force=False)
                            logger.info("%s deployment deletion return %s %s", log_prefix, rc_del, result_del)
                            termination_deletion_issued += 1
                    else:
                        logger.info("%s interrupted", log_prefix)
                        ret = AXWorkflowNodeResult.INTERRUPTED, None
                        break
                    logger.info("%s workflow in RUNNING_DEL state", log_prefix)
                else:
                    logger.info("%s workflow in RUNNING state", log_prefix)

                # check whether the fixture need to be terminated
                if node.is_fixture:
                    if node.fixtures_need_to_be_terminated:
                        logger.info("%s fixture_terminated", log_prefix)
                        if local_is_expecting:
                            # force deleting container, need no callback
                            rc_del, result_del = axsys_client.delete_service(service_name=uc_name, force=True)
                            logger.info("%s force delete return %s %s", log_prefix, rc_del, result_del)
                            ret = AXWorkflowNodeResult.SUCCEED, {AXWorkflowNodeResult.DETAIL_TAG_FIXTURE_TERMINATED_BEFORE_LAUNCH: True}
                            break
                        else:
                            if node.is_dind:
                                rc_del, result_del = axsys_client.delete_service(service_name=uc_name, force=False)
                            else:
                                rc_del, result_del = axsys_client.delete_service(service_name=uc_name, delete_pod=False,
                                                                                 stop_running_pod_only=True if termination_deletion_issued < MAX_TERMINATION_DELETION_ISSUED else False)
                            termination_deletion_issued += 1
                            logger.info("%s deletion return %s %s. termination_deletion_issued %s", log_prefix, rc_del, result_del, termination_deletion_issued)
                    else:
                        logger.info("%s fixture not terminated", log_prefix)

                # check whether the container has been launched
                if local_is_expecting:
                    cont_launch, _ = axdb_client.get_workflow_kv(launch_key)
                    if cont_launch:
                        container_launch_ack(cont_launch)
                    else:
                        logger.info("%s not launched yet, %s", log_prefix, launch_key)

                # fast path to try to get container result
                container_result, _ = axdb_client.get_workflow_kv(task_result_key)
                if container_result:
                    ret = process_redis_task_result(container_result)
                    if ret[0]:
                        logger.info("%s redis result=%s fast path",
                                    log_prefix, container_result)
                        got_result = True
                        break

                logger.info("%s no redis result yet %s", log_prefix, task_result_key)

                container_status = axsys_client.get_container_status(container_name=uc_name)
                logger.info("%s container status=%s", log_prefix, container_status)
                need_to_delete_container = False
                if container_status not in [AxsysClient.CONTAINER_RUNNING, axsys_client.CONTAINER_IMAGE_PULL_BACKOFF, AxsysClient.CONTAINER_PENDING]:
                    node.first_time_see_image_pull = None
                    if container_status in [AxsysClient.CONTAINER_STOPPED, AxsysClient.CONTAINER_FAILED]:
                        if local_is_expecting:
                            logger.warning("%s this is strange", log_prefix)
                        need_to_delete_container = True
                        logger.info("%s: delay delete, force=%s", log_prefix, local_is_expecting)
                    else:
                        logger.info("%s: container in %s", log_prefix, container_status)

                    # check result
                    container_result, _ = axdb_client.get_workflow_kv(task_result_key)
                    if container_result:
                        ret = process_redis_task_result(container_result)
                        if ret[0]:
                            logger.info("%s callback result=%s",
                                        log_prefix, container_result)
                            got_result = True
                            break

                    logger.debug("%s no redis result yet", log_prefix)
                    # no result, not running
                    if node.is_fixture and node.fixtures_termination_triggered:
                        ret = AXWorkflowNodeResult.FAILED, {
                            AXWorkflowNodeResult.DETAIL_TAG_FAILURE_REASON: AXWorkflowNodeResult.FAILED_FORCE_TERMINATE}
                        logger.warning("strange, %s terminate fixture container", log_prefix)
                        break
                    if need_to_delete_container:
                        # delayed delete
                        logger.info("%s: (delayed) delete stopped, force=%s", log_prefix, local_is_expecting)
                        rc_del, result_del = axsys_client.delete_service(service_name=uc_name,
                                                                         force=local_is_expecting)
                        logger.info("%s (delayed) delete return %s %s", log_prefix, rc_del, result_del)

                    if container_has_started:
                        # container crashed?
                        logger.info("%s no container running and no result (%s retry left)",
                                    log_prefix, missing_container_retry_left)
                        missing_container_retry_left -= 1
                        if missing_container_retry_left < 0:
                            ret = AXWorkflowNodeResult.FAILED, {AXWorkflowNodeResult.DETAIL_TAG_FAILURE_REASON: AXWorkflowNodeResult.FAILED_LOST_CONTAINER}
                            logger.critical("%s no container running and no result. container is missing? %s",
                                            log_prefix, missing_container_retry_left)
                            break
                    else:
                        logger.info("%s LAUNCH service_template=%s",
                                    log_prefix, pprint.pformat(service_template))

                        try:
                            logger.info("%s Create service payload=%s", log_prefix, pprint.pformat(service_template))
                            rc, containers, response_status_code, response_json = axsys_client.create_service(service_template)
                        except (requests.ConnectionError, requests.Timeout) as e:
                            logger.exception("%s create_service timeout=%s", log_prefix)
                            self.record_last_exception_event(exception=e)
                            logger.error("%s os._exit(12). will restart", log_prefix)
                            os._exit(12)
                        except Exception:
                            logger.exception("%s bad service_template=%s",
                                             log_prefix, node.service_template)
                            return AXWorkflowNodeResult.FAILED, {
                                AXWorkflowNodeResult.DETAIL_TAG_FAILURE_REASON: AXWorkflowNodeResult.FAILED_BAD_TEMPLATE}

                        if rc:
                            remote_container_name = axsys_client.canonical_container_full_name(containers[0])
                            logger.info("%s %s created", log_prefix, remote_container_name)
                            assert remote_container_name == uc_name, "{} bad name {} vs {}.".format(
                                log_prefix, remote_container_name, uc_name)
                        else:
                            logger.info("%s negative return from create_service, check axmon again", log_prefix)
                            container_status = axsys_client.get_container_status(container_name=uc_name)
                            if container_status not in [AxsysClient.CONTAINER_RUNNING, axsys_client.CONTAINER_IMAGE_PULL_BACKOFF, AxsysClient.CONTAINER_PENDING]:
                                logger.warning("%s not running (%s), check redis result again.",
                                               log_prefix, container_status)
                                container_result, _ = axdb_client.get_workflow_kv(task_result_key)
                                if container_result:
                                    ret = process_redis_task_result(container_result)
                                    if ret[0]:
                                        logger.warning("%s got callback result=%s after RECHECK",
                                                       log_prefix, container_result)
                                        got_result = True
                                        break

                                logger.warning("%s no redis result in RECHECK", log_prefix)

                                logger.critical("%s cannot launch container (%s) response_status=%s response_json=%s. return FAILED",
                                                log_prefix, container_status, response_status_code, response_json)
                                ret = AXWorkflowNodeResult.FAILED, \
                                      {AXWorkflowNodeResult.DETAIL_TAG_FAILURE_REASON: AXWorkflowNodeResult.FAILED_CANNOT_LAUNCH_CONTAINER,
                                       "response_status_code": response_status_code,
                                       AXWorkflowNodeResult.DETAIL_TAG_FAILURE_MESSAGE: json.dumps(response_json)}
                                break
                            else:
                                logger.warning("%s is already %s. (this should rarely happen)",
                                               log_prefix, container_status)
                elif container_status == axsys_client.CONTAINER_IMAGE_PULL_BACKOFF:
                    if not node.first_time_see_image_pull:
                        logger.warning("%s in %s. first time", log_prefix, container_status)
                        node.first_time_see_image_pull = time.time()
                    else:
                        time_now = time.time()
                        if time_now > node.first_time_see_image_pull + 60 * 5:
                            logger.warning("%s in %s pull image timeout.", log_prefix, container_status)
                            ret = AXWorkflowNodeResult.FAILED, \
                                  {AXWorkflowNodeResult.DETAIL_TAG_FAILURE_REASON: AXWorkflowNodeResult.FAILED_CANNOT_PULL_IMAGE}
                            break
                        else:
                            logger.warning("%s in %s (%s, %s). wait more time", log_prefix, container_status, node.first_time_see_image_pull, time_now)
                    node.report_result_to_kafka()
                else:
                    node.first_time_see_image_pull = None

                container_has_started = True

            elif workflow.status in [AXWorkflow.DELETED]:
                logger.warning("%s interrupted, but state is already %s. This should not happen",
                               log_prefix, AXWorkflow.DELETED)
                ret = AXWorkflowNodeResult.INTERRUPTED, None
                break

            elif workflow.status in [AXWorkflow.RUNNING_DEL_FORCE]:
                logger.info("%s force interrupted", log_prefix)
                ret = AXWorkflowNodeResult.INTERRUPTED, None
                break
            else:
                assert False, "{} Invalid status from DB, this should never happen".format(log_prefix)

            ret1 = wait_container_status_change()
            if not ret1:
                logger.info("%s is still running", log_prefix)
            else:
                logger.info("%s got something from check_status", log_prefix)

        container_status = axsys_client.get_container_status(container_name=uc_name)
        if container_status in [AxsysClient.CONTAINER_RUNNING, axsys_client.CONTAINER_IMAGE_PULL_BACKOFF, AxsysClient.CONTAINER_PENDING,
                                AxsysClient.CONTAINER_STOPPED, AxsysClient.CONTAINER_FAILED, AxsysClient.CONTAINER_UNKNOWN]:
            if container_status in [AxsysClient.CONTAINER_RUNNING, axsys_client.CONTAINER_IMAGE_PULL_BACKOFF, AxsysClient.CONTAINER_PENDING, AxsysClient.CONTAINER_UNKNOWN]:
                if ret[0] == AXWorkflowNodeResult.INTERRUPTED:
                    logger.info("%s is still %s, stop", log_prefix, container_status)

            logger.info("%s final delete, status=%s force=%s", log_prefix, container_status, local_is_expecting)

            keep_pod_condition = list([AXWorkflowNodeResult.FAILED_FORCE_TERMINATE, AXWorkflowNodeResult.FAILED_LOST_CONTAINER,
                                       AXWorkflowNodeResult.FAILED_CANNOT_FIND_RETURN])
            keep_pod_condition.append(AXWorkflowNodeResult.FAILED_SIG_TERM)

            # enable keep-bad-pod-arround
            if (not local_is_expecting) and (not node.is_dind) and ret[0] == AXWorkflowNodeResult.FAILED and ret[1].get(AXWorkflowNodeResult.DETAIL_TAG_FAILURE_REASON, None) in keep_pod_condition:
                # keep the pod for debugging purpose
                logger.info("keep the pod for debugging purpose. container_status=%s", container_status)
                rc_del, result_del = axsys_client.delete_service(uc_name, delete_pod=False, force=got_result)
            else:
                rc_del, result_del = axsys_client.delete_service(uc_name, force=local_is_expecting or got_result)
            logger.info("%s deletion return %s %s", log_prefix, rc_del, result_del)

        return ret

    def _start_and_monitor_container_thread(self, node):
        try:
            result_code, detail = self._start_and_monitor_container(node=node, root_service_template=self._service_template)
            logger.info('%s got result_code %s', node.log_prefix, result_code)

            self._add_result_to_q(node_id=node.node_id, name=node.name, result_code=result_code, detail=detail)
            logger.info('%s result enqueued', node.log_prefix)
        except Exception as e:
            logger.exception("got exception")
            sleep_second = 20
            logger.info("%s exception. sleep %s seconds", node.log_prefix, sleep_second)
            time.sleep(sleep_second)

            self.record_last_exception_event(exception=e)
            logger.error("%s os._exit(2). will restart", node.log_prefix)
            os._exit(2)

    def do_report_to_kafka(self, leaf_service_id, payload):
        if not self._fake_run:
            self.really_do_report_to_kafka(workflow_id=self.workflow_id, payload=payload, leaf_service_id=leaf_service_id)

    @staticmethod
    def do_report_to_node_event(workflow_id, leaf_id, payload):
        logger.info("Report to Node Events sid=%s in Axdb, %s", leaf_id, json.dumps(payload, indent=2))
        if leaf_id is None:
            leaf_id = workflow_id
        if payload:
            timestamp = payload.get('start_date', None) or payload.get('end_date', None) \
                        or AXWorkflow.get_current_epoch_timestamp_in_ms()
            axdb_client.create_node_event(root_id=workflow_id, leaf_id=leaf_id, result=payload['status'],
                                          timestamp=timestamp, status_detail=payload['status_detail']['code'],
                                          detail=json.dumps(payload.get('detail', None)))

    @staticmethod
    def really_do_report_to_kafka(workflow_id, payload, leaf_service_id=None):
        try:
            wid = uuid.UUID(workflow_id)
            # axops uses uuid1() to generate workflow_id while workflow tests use uuid4()
            if wid.version == 1:
                ExecutorProducerClient.send_executor_status(key=workflow_id, payload=payload)
            else:
                logger.warning("Not a uuid1, not send event for %s %s", workflow_id, leaf_service_id)
        except Exception:
            logger.exception("Not a uuid %s, not send event for %s %s", workflow_id, leaf_service_id)

        AXWorkflowExecutor.do_report_to_node_event(workflow_id=workflow_id, leaf_id=leaf_service_id, payload=payload)

    def get_node(self, node_id):
        return self._nodes[node_id]

    @staticmethod
    def report_workflow_scheduled_to_kafka(workflow_id):
        states = {"service_id": workflow_id,
                  "status": "WAITING",
                  "result": ExecutorProducerClient.WAITING_STATE,
                  "status_detail": {'code': "TASK_SCHEDULED"},
                  "end_date": AXWorkflow.get_current_epoch_timestamp_in_ms(),
                  "run_duration": 0
        }
        AXWorkflowExecutor.really_do_report_to_kafka(workflow_id=workflow_id, payload=states)
        logger.info("%s reported scheduled to axops", AXWorkflowExecutor.get_log_prefix(workflow_id))

    @staticmethod
    def report_workflow_cancel_to_kafka(workflow_id):
        states = {"service_id": workflow_id,
                  "status": "COMPLETE",
                  "result": ExecutorProducerClient.CANCELLED_RESULT,
                  "status_detail": {'code': "TASK_CANCELLED"},
                  "end_date": AXWorkflow.get_current_epoch_timestamp_in_ms(),
                  "run_duration": 0
        }
        AXWorkflowExecutor.really_do_report_to_kafka(workflow_id=workflow_id, payload=states)
        logger.info("%s reported cancel to axops", AXWorkflowExecutor.get_log_prefix(workflow_id))

    @staticmethod
    def report_workflow_force_termination_to_kafka(workflow_id):
        states = {"service_id": workflow_id,
                  "status": "COMPLETE",
                  "result": ExecutorProducerClient.FAILURE_RESULT,
                  "status_detail": {'code': "FORCE_TERMINATED"},
                  "end_date": AXWorkflow.get_current_epoch_timestamp_in_ms(),
                  "run_duration": 0
        }
        AXWorkflowExecutor.really_do_report_to_kafka(workflow_id=workflow_id, payload=states)
        logger.info("%s reported force termination to axops", AXWorkflowExecutor.get_log_prefix(workflow_id))

    def force_fail_workflow(self):
        self.report_workflow_force_termination_to_kafka(self._workflow_id)
        logger.info("%s wait for proper state before call last step", self.log_prefix)
        self._wait_for_state_to_be_running_or_running_del_or_done()
        time.sleep(1)
        AXWorkflowEvent.save_workflow_event_to_db(workflow_id=self.workflow_id,
                                                  event_type=AXWorkflowEvent.FORCE_TERMINATE)
        logger.info("%s ready to call last_step", self.log_prefix)
        self.last_step(AXWorkflowNodeResult.FAILED, forced=True)
        assert False, "should never reach here"

    def record_last_exception_event(self, exception):
        with self._last_exception_recoded_lock:
            if not self._last_exception_recoded:
                self._last_exception_recoded = True
                count = 0
                max_retry = 60
                while True:
                    count += 1
                    try:
                        AXWorkflowEvent.save_workflow_exception_event_to_db(workflow_id=self.workflow_id, exception=exception)
                        return
                    except Exception:
                        logger.exception("cannot save exception")
                        if count < max_retry:
                            time.sleep(5)
                        else:
                            logger.warning("cannot save exception")
                            return

    def check_previous_events(self):
        exceptions_threshold = 20
        total_exceptions_threshold = 50
        consecutive_exceptions_interval_max = 60 * 1000

        events = self._get_workflow_events_from_db()
        consecutive_exceptions = 0
        max_consecutive_exceptions = 0
        total_exceptions = 0
        last_exception_ts = 0
        # simple exception history check
        for event in events:
            logger.info("%s event: %s", self.log_prefix, event.jsonify())
            if event.event_type == AXWorkflowEvent.EXCEPTION:
                total_exceptions += 1
                if (event.timestamp - last_exception_ts < consecutive_exceptions_interval_max and event.timestamp > last_exception_ts) \
                        or last_exception_ts == 0:
                    consecutive_exceptions += 1
                    if max_consecutive_exceptions < consecutive_exceptions:
                        max_consecutive_exceptions = consecutive_exceptions
                else:
                    consecutive_exceptions = 1
                last_exception_ts = event.timestamp
            elif event.event_type == AXWorkflowEvent.START:
                self._total_session += 1

        logger.info("%s total_session=%s total_exceptions=%s max_consecutive_exceptions=%s",
                    self.log_prefix, self._total_session, total_exceptions, max_consecutive_exceptions)

        if max_consecutive_exceptions > exceptions_threshold:
            logger.critical("%s too many consecutive exceptions %s > %s",
                            self.log_prefix, max_consecutive_exceptions, exceptions_threshold)
            self.force_fail_workflow()

        if total_exceptions > total_exceptions_threshold:
            logger.critical("%s too many exceptions %s > %s",
                            self.log_prefix, total_exceptions, total_exceptions_threshold)
            self.force_fail_workflow()

    def run(self):
        try:
            self.init()
            self._start_heartbeat_thread()
            self._send_heartbeat = True

            AXWorkflowEvent.save_workflow_event_to_db(workflow_id=self.workflow_id,
                                                      event_type=AXWorkflowEvent.START,
                                                      detail={"version": __version__})
            self.check_previous_events()
            try:
                # build nodes
                self._build_nodes()
            except Exception as e:
                logger.exception("%s cannot build nodes", self.log_prefix)
                AXWorkflowEvent.save_workflow_exception_event_to_db(workflow_id=self.workflow_id, exception=e)
                self.force_fail_workflow()

            self._recover()
            self._can_send_nodes_status = True
            if self._test_crash_second:
                self._start_crash_thread(max_crash_second=self._test_crash_second)
            self._start_if_have_not()
            self._wait_and_process_results()
        except Exception as e:
            sleep_second = 20
            logger.exception("got exception. sleep %s seconds", sleep_second)
            time.sleep(sleep_second)
            self.record_last_exception_event(exception=e)
            logger.error("%s os._exit(1). will restart", self.log_prefix)
            os._exit(1)

        logger.info("%s: exit run()", self.log_prefix)

    @staticmethod
    def _wait_x_second_then_crash_thread(workflow_id, max_crash_second):
        global global_test_mode
        global_test_mode = True
        max_crash_second_lower_bound = 100

        if max_crash_second < max_crash_second_lower_bound:
            logger.info("%s override max_crash_second from %s to %s",
                        AXWorkflowExecutor.get_log_prefix(workflow_id), max_crash_second, max_crash_second_lower_bound)
        max_crash_second = max_crash_second_lower_bound

        crash_second = random.randint(0, max_crash_second)
        logger.info("%s will sleep %s/%s seconds and do crash test",
                    AXWorkflowExecutor.get_log_prefix(workflow_id), crash_second, max_crash_second)
        time.sleep(crash_second)
        logger.info("%s slept %s/%s seconds. time to do crash test. Call os._exit(10). will restart",
                    AXWorkflowExecutor.get_log_prefix(workflow_id), crash_second, max_crash_second)
        os._exit(10)

    def _start_crash_thread(self, max_crash_second):
        logger.info("%s start test-crash-thread", self.log_prefix)
        t = threading.Thread(name="test-crash-thread",
                             target=self._wait_x_second_then_crash_thread,
                             kwargs={'workflow_id': self.workflow_id,
                                     'max_crash_second': max_crash_second})
        t.daemon = True
        t.start()

    def _heartbeat_with_adc_thread(self, workflow_id):
        sleep_ms_at_least = 20 * 1000  # every 20 seconds
        sleep_ms_at_most = sleep_ms_at_least + 10 * 1000  # plus 10 seconds

        count = 0
        while True:
            sleep_second = int(random.randint(sleep_ms_at_least, sleep_ms_at_most) / 1000)
            while sleep_second > 0:
                time_start = time.time()
                try:
                    workflow_query_key = AXWorkflow.REDIS_QUERY_LIST_KEY.format(workflow_id)
                    tuple_val = redis_client.brpop([workflow_query_key], timeout=sleep_second)
                    if tuple_val is not None:
                        # collect the node info and send back
                        post_message = {"workflow_id": workflow_id,
                                        "event": "workflow_info"}
                        if self._can_send_nodes_status:
                            logger.info("%s collect workflow_info", self.log_prefix)
                            nodes = self._root_node.d3_format()
                        else:
                            nodes = {}
                        post_message["nodes"] = nodes
                        logger.info("%s send workflow_info", self.log_prefix)
                        requests.post(self._report_done_url, json=post_message)
                        break
                except Exception:
                    count += 1
                    logger.exception("%s %s exception while wait to send keep alive", self.log_prefix, count)
                    time.sleep(random.randint(3, 10))

                if self._send_heartbeat:
                    # to avoid sending a storm of request to adc when redis is down
                    time_end = time.time()
                    time_elapsed = time_end - time_start
                    if time_elapsed + 0.1 >= sleep_second or time_elapsed < 0:
                        try:
                            post_message = {"workflow_id": workflow_id,
                                            "event": "heartbeat",
                                            "resource": self._root_node.max_resource.toJson()}
                            logger.info("%s send heartbeat", self.log_prefix)
                            requests.post(self._report_done_url, json=post_message)
                        except Exception:
                            count += 1
                            logger.exception("%s %s exception while sending keep alive",
                                             self.log_prefix, count)
                            time.sleep(random.randint(3, 10))
                        break
                    else:
                        sleep_second = int(sleep_second - time_elapsed)
                else:
                    break

    def _start_heartbeat_thread(self):
        logger.info("%s start heartbeat-thread", self.log_prefix)
        t = threading.Thread(name="heartbeat-thread",
                             target=self._heartbeat_with_adc_thread,
                             kwargs={'workflow_id': self.workflow_id})
        t.daemon = True
        t.start()

    def start_start_and_monitor_container_thread(self, node, part_of_recover=False):
        if node.is_fixture:
            additional_log = "fixtures_termination_triggered={} fixtures_need_to_be_terminated={}".\
                format(node.fixtures_termination_triggered, format(node.fixtures_need_to_be_terminated))
        else:
            additional_log = ""

        logger.info("%s [recover=%s] start monitor container thread. type=%s state=%s %s",
                    node.log_prefix, part_of_recover,
                    node.get_type_string(), node.state, additional_log)
        t = threading.Thread(name=node.get_type_string() + "-worker-" + node.node_id,
                             target=self._start_and_monitor_container_thread, args=(node,))
        t.daemon = True
        t.start()

    def report_to_adc(self, result_code, last_status, nodes_stats, max_retry=1000):
        if self._report_done_url:
            counter = 0
            post_message = {"workflow_id": self.workflow_id,
                            "event": "done",
                            "last_status": last_status,
                            "nodes_stats": nodes_stats}
            while counter < max_retry:
                counter += 1
                try:
                    if self._test_expected_failure_node is not None:
                        msg = None
                        if last_status == AXWorkflow.SUCCEED and self._test_expected_failure_node > 0:
                            msg = "test expect {} nodes fail, but workflow result is {}".format(self._test_expected_failure_node,
                                                                                                result_code)
                        elif (last_status == AXWorkflow.FAILED or last_status == AXWorkflow.FORCED_FAILED) and self._test_expected_failure_node == 0:
                            msg = "test expect {} nodes fail, but workflow result is {}".format(
                                self._test_expected_failure_node,
                                result_code)
                        if msg:
                            logger.critical("%s", msg)
                            post_message["CRITICAL"] = msg

                        post_message["expected_failure_node"] = self._test_expected_failure_node

                    logger.info("%s report to %s. last_status=%s result=%s",
                                self.log_prefix,
                                self._report_done_url,
                                last_status,
                                post_message)
                    r = requests.post(self._report_done_url, json=post_message)
                    if r.status_code == 200:
                        logger.info("%s report to %s. done",
                                    self.log_prefix,
                                    self._report_done_url)
                        return
                    else:
                        logger.info("%s report to %s return %s %s. retry %s/%s",
                                    self.log_prefix,
                                    self._report_done_url,
                                    r.status_code, r.json(),
                                    counter, max_retry)
                        continue
                except requests.ConnectionError:
                    logger.warning("reporting connect... retry %s/%s", counter, max_retry)
                    time.sleep(10)
                except Exception:
                    logger.exception("reporting connect. retry %s/%s", counter, max_retry)
                    time.sleep(10)

            logger.error("%s cannot report to %s. result=%s", self.log_prefix,
                         self._report_done_url, post_message)
            return
        else:
            logger.info("%s no report", self.log_prefix)

    def last_step_update_db(self, result_code, forced):
        count = 0
        if forced:
            assert result_code == AXWorkflowNodeResult.FAILED
        while True:
            workflow = AXWorkflow.get_workflow_by_id_from_db(self._workflow_id)
            assert workflow
            if workflow.status in [AXWorkflow.RUNNING_DEL, AXWorkflow.RUNNING_DEL_FORCE]:
                new_status = AXWorkflow.DELETED
            elif workflow.status in [AXWorkflow.RUNNING]:
                if result_code == AXWorkflowNodeResult.SUCCEED:
                    new_status = AXWorkflow.SUCCEED
                elif result_code == AXWorkflowNodeResult.FAILED:
                    if forced:
                        new_status = AXWorkflow.FORCED_FAILED
                    else:
                        new_status = AXWorkflow.FAILED
                else:
                    assert False, "{} bad state {} last result_code={}".format(self.log_prefix, workflow.status, result_code)
            elif workflow.status in [AXWorkflow.DELETED, AXWorkflow.SUCCEED, AXWorkflow.FAILED, AXWorkflow.FORCED_FAILED]:
                logger.info("%s no change workflow status %s", self.log_prefix, workflow.status)
                return workflow.status
            else:
                assert False, "{} bad state {}, last result_code={}".format(self.log_prefix, workflow.status, result_code)

            assert new_status is not None
            logger.info("%s change workflow status %s->%s", self.log_prefix, workflow.status, new_status)
            if AXWorkflow.update_workflow_status_in_db(workflow=workflow, new_status=new_status):
                return new_status
            else:
                count += 1
                if count > 10:
                    assert False, "{} state {} too many update failure. last result_code={}".format(self.log_prefix, workflow.status, result_code)

    def last_step(self, result_code, forced):
        if self._fake_run:
            return

        nodes_stats = self._get_nodes_stats()
        logger.info("%s after last_step %s", self.log_prefix, nodes_stats)
        logger.info("%s last_step result=%s", self.log_prefix, result_code)
        logger.info("%s Workflow Tree after last step \n%s", self.log_prefix, str(self._root_node))

        last_status = self.last_step_update_db(result_code=result_code, forced=forced)
        self.report_to_adc(result_code=result_code, last_status=last_status, nodes_stats=nodes_stats)
        AXWorkflowEvent.save_workflow_event_to_db(workflow_id=self._workflow_id,
                                                  event_type=AXWorkflowEvent.TERMINATE,
                                                  detail={"result_code": result_code,
                                                          "last_status": last_status,
                                                          "nodes_stats": nodes_stats})
        self._send_heartbeat = False
        self._can_send_nodes_status = False
        self.shutdown()

    def stop_self_container(self, max_retry=30):
        count = 0
        while True:
            logger.info("%s kill self_container %s", self.log_prefix, self._self_container_name)
            # delete_service may stuck or fail
            axsys_client.delete_service(self._self_container_name)
            time.sleep(60)
            count += 1
            if count > max_retry:
                logger.info("%s call os.exit(9)", self.log_prefix)
                os._exit(9)

    def shutdown(self):
        with self._results_q_cond:
            self._shutdown = True
            self._results_q_cond.notifyAll()
            logger.info("%s sent shutdown notification", self.log_prefix)

        self.stop_self_container()

    def _get_workflow_results_from_db(self):
        return AXWorkflowNodeResult.load_results_from_db(self._workflow_id)

    def _get_workflow_events_from_db(self):
        return AXWorkflowEvent.load_events_from_db(workflow_id=self._workflow_id)

    def _get_workflow_node_events_from_db(self):
        events = axdb_client.get_node_events(root_id=self.workflow_id)
        results = dict()
        for event in events:
            leaf_id = event.get('leaf_id', None)
            if leaf_id not in results:
                results[leaf_id] = list()
            results[leaf_id].append(event)
        return results

    def check_event_already_sent(self, leaf_id, status):
        if self._node_events and leaf_id in self._node_events:
            for event in self._node_events[leaf_id]:
                if event.get('result', None) == status:
                    return True
        return False
