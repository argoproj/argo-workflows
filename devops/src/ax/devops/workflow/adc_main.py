#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2016 Applatix, Inc. All rights reserved.
#

"""
Module for ADC / AdmissionController
"""
import copy
import json
import logging
import math
import numbers
import os
import pprint
import threading
import time
import traceback
import uuid

from six import with_metaclass
from multiprocessing.pool import ThreadPool

from ax.devops.axdb.axdb_client import AxdbClient
from ax.devops.axsys.axsys_client import AxsysClient
from ax.devops.redis.redis_client import RedisClient, DB_RESULT
from ax.devops.kafka.kafka_client import EventNotificationClient
from ax.devops.utility.axworkflow import substitute_parameters

from ax.exceptions import AXIllegalOperationException, AXServiceTemporarilyUnavailableException, \
    AXIllegalArgumentException, AXWorkflowAlreadyFailed, AXWorkflowAlreadySucceed, AXWorkflowDoesNotExist
from ax.version import __version__
from ax.util.singleton import Singleton
from ax.util.ax_signal import traceback_multithread

from .adc_state import ADCStateMachine, ADCState
from .ax_workflow import AXWorkflow, AXWorkflowResource, AXResource
from .ax_workflow_constants import INSTANCE_RESOURCE, AX_EXECUTOR_RESOURCE
from .ax_workflow_executor import AXWorkflowExecutor, AXWorkflowNodeResult, AXWorkflowEvent, WorkflowNode
from .test_ax_workflow_executor import gen_random_workflow
from ax.notification_center import FACILITY_AX_WORKFLOW_ADC, CODE_ADC_MISSING_HEARTBEAT_FROM_WFE

logger = logging.getLogger(__name__)

axsys_client = AxsysClient()
axdb_client = AxdbClient()
redis_client = RedisClient(host='redis.axsys', db=DB_RESULT, retry_max_attempt=360, retry_wait_fixed=5000)
event_notification_client = EventNotificationClient(FACILITY_AX_WORKFLOW_ADC)

ADC_DEFAULT_PORT = 8911
ADC_WORKFLOW_ID = "1"


class ADC(with_metaclass(Singleton, object)):
    def __init__(self):
        super(ADC, self).__init__()
        self.version = __version__
        self._shutdown = False

        self._workflow_sets_mutex = threading.Lock()
        self._workflow_sets = {}

        self._suspended_q = []
        self._suspended_map = {}
        self._suspended_q_cond = threading.Condition()
        self._admitted_q = []
        self._admitted_q_cond = threading.Condition()

        # Load resource info from cluster config before start
        # If cannot get the config, adc will fail
        cluster_config = axsys_client.get_cluster_config()
        ax_reserved_cpu = float(cluster_config["user_node_resource_rsvp"]["cpu"])
        ax_reserved_mem = float(cluster_config["user_node_resource_rsvp"]["memory"])
        self.instance_sys_resource = [ax_reserved_cpu / 1000.0, ax_reserved_mem]
        self.max_auto_scale = int(cluster_config['max_node_count']) - int(cluster_config['axsys_node_count'])
        self.max_vol_size = float(cluster_config['ax_vol_size'])
        self.instance_type = cluster_config['axuser_node_type']

        logger.info("[adc]: Argo system reserved resource (CPU / Memory): %s", self.instance_sys_resource)
        logger.info("[adc]: User node instance type: %s", self.instance_type)
        logger.info("[adc]: Max autoscale: %s", self.max_auto_scale)
        logger.info("[adc]: Max vol size: %s", self.max_vol_size)

        if self.instance_type not in INSTANCE_RESOURCE.keys():
            logger.error("[adc]: MINION_NODE_TYPE cannot find: %s, using m3.large as default", self.instance_type)
            self.instance_type = 'm3.large'

        node_resource = INSTANCE_RESOURCE[self.instance_type]
        self._instance_resource = AXWorkflowResource(node_resource) - AXWorkflowResource(self.instance_sys_resource)
        self._total_resource = AXWorkflowResource([x * self.max_auto_scale for x in self._instance_resource.resource])
        self._used_resource = AXWorkflowResource()
        self._executor_resource = AXWorkflowResource(AX_EXECUTOR_RESOURCE)

        # Calculate ratio
        ratio = 1.0 - max(self.instance_sys_resource[0]/float(node_resource[0]),
                          self.instance_sys_resource[1]/float(node_resource[1]))
        # Round down for available ratio
        self._ax_sys_resource_ratio = math.floor(ratio * 100) / 100.0

        # Track list of workflows that currently reserve resources
        self._resource_reserving_workflow_set = dict()

        # Track list of resources that currently being reserved
        self._resource_reserving_resource_set = dict()
        self._resource_reserving_intermediate_set = set()

        self._port = None

        self.asm = ADCStateMachine()

        self._wfe_registry = None
        self._wfe_namespace = None
        self._wfe_version = None

        my_path = os.path.dirname(os.path.abspath(__file__))
        template_file = os.path.join(my_path, "axworkflowexecutor.json")
        with open(template_file, 'r') as data:
            self._service_template = json.load(data)

        self._workflows_info = {}

        # Lock for notification center alerts
        self._notification_center_lock = threading.Lock()
        self._notification_center_list = dict()

        self._revive_delete_lock = threading.Lock()
        self._revive_delete_map = {}

        self._worker_thread_lock = threading.Lock()
        self._worker_thread_total = 0


    def set_param(self, wfe_registry, wfe_namespace, wfe_version):
        self._wfe_registry = wfe_registry
        self._wfe_namespace = wfe_namespace
        self._wfe_version = wfe_version

    def set_port(self, port):
        self._port = port

    @property
    def state(self):
        return self.asm.state

    def get_state(self):
        state = self.state

        if state in [ADCState.UNKNOWN, ADCState.STARTING, ADCState.STOPPED]:
            available_states = []
        elif self.state in [ADCState.RUNNING, ADCState.SUSPENDED_ALLOW_NEW, ADCState.SUSPENDED_NO_NEW]:
            available_states = [ADCState.RUNNING, ADCState.SUSPENDED_ALLOW_NEW,
                                ADCState.SUSPENDED_NO_NEW, ADCState.STOPPED]
        else:
            available_states = []

        ret = {"state": state}

        if available_states:
            ret["available_states"] = available_states

        return ret

    def request_set_state(self, new_state):
        logger.info("[adc]: request new_state=%s, current_state=%s", new_state, self.state)
        msg = "[adc]: Cannot change ADC state to {} when current state is {}".format(new_state, self.state)
        msg2 = "[adc]: Cannot change ADC state to {}".format(new_state)

        if self.state in [ADCState.UNKNOWN, ADCState.STARTING, ADCState.STOPPED]:
            raise AXServiceTemporarilyUnavailableException(msg)
        elif self.state in [ADCState.RUNNING, ADCState.SUSPENDED_NO_NEW, ADCState.SUSPENDED_ALLOW_NEW]:
            pass
        else:
            raise AXIllegalOperationException(msg)

        if new_state in [ADCState.UNKNOWN, ADCState.STARTING]:
            raise AXIllegalArgumentException(msg2)
        elif new_state in [ADCState.STOPPED]:
            # restart ADC
            self._do_shutdown()
        elif new_state in [ADCState.RUNNING, ADCState.SUSPENDED_NO_NEW, ADCState.SUSPENDED_ALLOW_NEW]:
            if new_state == ADCState.RUNNING:
                self.asm.request_running()
            elif new_state == ADCState.SUSPENDED_NO_NEW:
                self.asm.request_suspended_no_new()
            elif new_state == ADCState.SUSPENDED_ALLOW_NEW:
                self.asm.request_suspended_allow_new()
            else:
                assert False
            self._notify_all_q()
        else:
            raise AXIllegalOperationException(msg2)

        return {}

    @staticmethod
    def startup_prerequisite():
        logger.info("[adc]: starting...")
        logger.info("[adc]: AXDB version: %s", AXWorkflow.get_db_version_wait_till_db_is_ready())
        redis_client.wait(timeout=30 * 60)
        logger.info("[adc]: redis is available")

    def signal_debugger(signal_num, frame):
        logger.info("ADC debugged with signal %s", signal_num)
        result = traceback_multithread(signal_num, frame)
        logger.info(result)

    def run(self):
        """ADC main thread"""
        import signal
        import sys

        def signal_handler(sig, frame):
            logger.info("[adc] killed with signal %s", sig)
            sys.exit(0)

        signal.signal(signal.SIGTERM, signal_handler)
        signal.signal(signal.SIGINT, signal_handler)
        signal.signal(signal.SIGUSR1, self.signal_debugger)
        try:
            self.asm.init_starting()
            self.save_workflow_start_event_to_db()
            self._recover()
            self.asm.done_starting()
            self._start_launcher_thread()
            self._start_heartbeat_thread()
            self._start_delete_resource_thread()

            self._admission_controller()
        except (KeyboardInterrupt, SystemExit):
            pass
        except Exception as e:
            sleep_second = 20
            logger.exception("got exception. sleep %s seconds", sleep_second)
            time.sleep(sleep_second)
            self.save_workflow_exception_event_to_db(exception=e)

        logger.info("[adc]: exiting")

    def _notify_all_q(self):
        with self._suspended_q_cond:
            self._suspended_q_cond.notifyAll()
        with self._admitted_q_cond:
            self._admitted_q_cond.notifyAll()

    def _do_shutdown(self):
        logger.info("[adc]: send shutdown notification")
        self.asm.shutdown()
        self._shutdown = True
        self._notify_all_q()

    def _update_workflow_sets(self, workflow, new_status):
        return self._update_workflow_sets_by_id(workflow_id=workflow.id, new_status=new_status)

    def _update_workflow_sets_by_id(self, workflow_id, new_status):
        timestamp = AXWorkflow.get_current_epoch_timestamp_in_ms()
        with self._workflow_sets_mutex:
            inserted = False
            for key, value in self._workflow_sets.items():
                assert isinstance(value, dict)
                if new_status == key:
                    value[workflow_id] = timestamp
                    inserted = True
                else:
                    # remove from old
                    if workflow_id in value:
                        value.pop(workflow_id)
            if not inserted and new_status is not None:
                self._workflow_sets[new_status] = {workflow_id: timestamp}

    def _refresh_workflow_sets_by_id(self, workflow_id):
        timestamp = AXWorkflow.get_current_epoch_timestamp_in_ms()
        which_key = []
        with self._workflow_sets_mutex:
            for status in [AXWorkflow.RUNNING, AXWorkflow.RUNNING_DEL, AXWorkflow.RUNNING_DEL_FORCE]:
                col = self._workflow_sets.get(status, {})
                if col and workflow_id in col:
                    col[workflow_id] = timestamp
                    return True

            # not found in RUNNING and RUNNING_DEL, check all
            for key, value in self._workflow_sets.items():
                if workflow_id in value:
                    which_key.append(key)

        msg = "[adc] [{}]: uncommon liveness update. in {} set.".format(workflow_id, which_key)
        logger.debug("%s", msg)
        return False

    def workflows_show(self, recent_seconds=0, verbose=False, url_root=None):
        logger.info("[adc] workflows_show: recent_seconds=%s verbose=%s", recent_seconds, verbose)
        id_set = set()
        ret = dict()

        used_resource_check = AXWorkflowResource()
        for value in self._resource_reserving_workflow_set.values():
            used_resource_check += AXWorkflowResource(value)
            used_resource_check += self._executor_resource

        # Compute resource check for the reserved resources
        temp_resource_reserving_resource = dict()
        for resource_id, resource in self._resource_reserving_resource_set.items():
            used_resource_check += resource.resource
            temp_resource_reserving_resource[resource_id] = resource.toJson()

        ret["resource"] = {
            'MINION_NODE_TYPE': self.instance_type,
            'MAX_CLUSTER_SIZE': self.max_auto_scale,
            'MAX_VOLUME_SIZE_GB': self.max_vol_size,
            'AX_NODE_MIN_RESERVED_RESOURCE': self.instance_sys_resource,
            'leftover_resource': (self._total_resource - self._used_resource).resource,
            'used_resource': self._used_resource.resource,
            'used_resource_check': used_resource_check.resource,
            'total_resource': self._total_resource.resource,
            'executor_resource': self._executor_resource.resource,
            'used_resource_workflow': self._resource_reserving_workflow_set,
            'amm_resource': temp_resource_reserving_resource,
            'minion_to_sys_resource_ratio': self._ax_sys_resource_ratio,
            'minion_available_resource': self._instance_resource.resource,
        }

        with self._workflow_sets_mutex:
            running_urls = []
            for key, value, in self._workflow_sets.items():
                if value:
                    if key in [AXWorkflow.RUNNING_DEL, AXWorkflow.RUNNING, AXWorkflow.RUNNING_DEL_FORCE]:
                        for wid in value:
                            running_urls.append("{}workflows_v/{}".format(url_root, wid))
                    ret[key] = list(value.keys())
                    id_set = id_set | set(value.keys())

        if running_urls:
            ret["RUNNING_GRAPHS"] = running_urls

        ret["total_unfinished"] = (id_set and len(id_set)) or 0
        if recent_seconds:
            timestamp_start = (AXWorkflow.get_current_epoch_timestamp_in_ms() - recent_seconds * 1000) * 1000
            events = [i.jsonify(no_nodes_stats=(not verbose)) for i in AXWorkflowEvent.load_events_from_db(timestamp_start=timestamp_start)]
            for event in events:
                if "workflow_id" in event:
                    event["adc_url"] = "{}workflows/{}".format(url_root,
                                                               event["workflow_id"])
                    if event["workflow_id"] != ADC_WORKFLOW_ID:
                        event["visual_url"] = "{}workflows_v/{}".format(url_root,
                                                                        event["workflow_id"])
            events.reverse()
            ret["recent_events"] = events

        return ret

    def workflows_delete(self):
        logger.info("[adc]: request delete ALL workflows state=%s", self.state)
        msg = "[adc]: Can only delete ALL workflows in state {}, while current state is {}.".format(ADCState.SUSPENDED_NO_NEW, self.state)

        if self.state in [ADCState.UNKNOWN, ADCState.STARTING, ADCState.STOPPED]:
            raise AXServiceTemporarilyUnavailableException(msg)
        elif self.state in [ADCState.RUNNING, ADCState.SUSPENDED_ALLOW_NEW]:
            raise AXIllegalOperationException(msg)
        elif self.state in [ADCState.SUSPENDED_NO_NEW]:
            pass
        else:
            raise AXIllegalOperationException(msg)

        id_set = set()
        with self._workflow_sets_mutex:
            for key, value, in self._workflow_sets.items():
                id_set = id_set | set(value.keys())

        self._workflows_delete_all_in_set(id_set=id_set)

        return self.workflows_show()

    def _workflows_delete_all_in_set(self, id_set):
        if not id_set:
            return

        results = []
        max_num_of_thread = 32
        num_of_workflow = len(id_set)
        pool = ThreadPool(min(num_of_workflow, max_num_of_thread))

        logger.info("[adc]: try delete %s workflows", num_of_workflow)
        for workflow_id in id_set:
            results += [pool.apply_async(self._workflow_delete_no_exception, (workflow_id, ))]
        pool.close()
        pool.join()
        logger.info("[adc]: done delete %s workflows", num_of_workflow)
        return

    def _recover(self):
        logger.info("[adc]: recover start")

        # get all SUSPENDED workflows:
        # - add to suspeneded
        logger.info("[adc]: load SUSPENDED")
        suspended_workflows = self._get_workflows_by_status_from_db(AXWorkflow.SUSPENDED)
        for workflow in suspended_workflows:
            self._add_suspended_workflow_to_q(workflow=workflow)

        # get all ADMITTED workflows:
        # - add to admitted queue
        # - reserve resource
        logger.info("[adc]: load ADMITTED")
        admitted_workflows = self._get_workflows_by_status_from_db(AXWorkflow.ADMITTED)
        for workflow in admitted_workflows:
            self._update_workflow_sets(workflow, AXWorkflow.ADMITTED)
            self._add_admitted_workflow_to_q(workflow=workflow)
            with self._suspended_q_cond:
                self._resource_reserve(workflow)

        # get all ADMITTED_DEL workflows:
        # - add to admitted queue
        # - reserve resource
        logger.info("[adc]: load ADMITTED_DEL")
        admitted_del_workflows = self._get_workflows_by_status_from_db(AXWorkflow.ADMITTED_DEL)
        for workflow in admitted_del_workflows:
            self._update_workflow_sets(workflow, AXWorkflow.ADMITTED_DEL)
            self._add_admitted_workflow_to_q(workflow=workflow)
            with self._suspended_q_cond:
                self._resource_reserve(workflow)

        # get all RUNNING workflows
        # - reserve resource
        logger.info("[adc]: load RUNNING")
        running_workflows = self._get_workflows_by_status_from_db(AXWorkflow.RUNNING)
        for workflow in running_workflows:
            self._update_workflow_sets(workflow, AXWorkflow.RUNNING)
            with self._suspended_q_cond:
                self._resource_reserve(workflow)

        # get all RUNNING_DEL workflows
        # - reserve resource
        logger.info("[adc]: load RUNNING_DEL")
        del_workflows = self._get_workflows_by_status_from_db(AXWorkflow.RUNNING_DEL)
        for workflow in del_workflows:
            self._update_workflow_sets(workflow, AXWorkflow.RUNNING_DEL)
            with self._suspended_q_cond:
                self._resource_reserve(workflow)

        # get all RUNNING_DEL workflows
        # - reserve resource
        logger.info("[adc]: load RUNNING_DEL_FORCE")
        del_workflows = self._get_workflows_by_status_from_db(AXWorkflow.RUNNING_DEL_FORCE)
        for workflow in del_workflows:
            self._update_workflow_sets(workflow, AXWorkflow.RUNNING_DEL_FORCE)
            with self._suspended_q_cond:
                self._resource_reserve(workflow)

        # get all resources from AXAMM
        logger.info("[adc]: load resources")
        resources = self._get_resources_from_db()
        for resource in resources:
            self._used_resource += resource.resource
            self._resource_reserving_resource_set[resource.resource_id] = resource
        logger.info("[adc]: recover done")

    def _admission_controller(self):
        def check_to_admit():
            if self._suspended_q[0].resource + self._executor_resource + self._used_resource <= self._total_resource:
                return True
            return False

        logger.info("[adc] [controller]: start")
        while not self._shutdown:
            picked = []
            with self._suspended_q_cond:
                if len(self._suspended_q) > 0 and self.state in [ADCState.RUNNING] and check_to_admit():
                    logger.debug("[adc] [controller]: no wait (running and has tasks and has resource)")
                elif self._shutdown:
                    logger.debug("[adc] [controller]: no wait (shutdown)")
                else:
                    logger.debug("[adc] [controller]: wait")
                    self._suspended_q_cond.wait()
                    logger.debug("[adc] [controller]: wakeup")
                while len(self._suspended_q) and self.state in [ADCState.RUNNING] and not self._shutdown:
                    workflow = self._suspended_q[0]
                    if not check_to_admit():
                        logger.info("[adc] [controller] [%s]: cannot admit. because %s + %s + %s > %s",
                                    workflow.id, workflow.resource, self._executor_resource, self._used_resource, self._total_resource)
                        break
                    logger.info("[adc] [controller] [%s]: admit. resource=%s", workflow.id, workflow.resource)
                    self._suspended_q.pop(0)
                    del self._suspended_map[workflow.id]
                    self._resource_reserve(workflow)
                    picked.append(workflow)

            # to avoid dead lock
            for workflow in picked:
                self._add_admitted_workflow_to_q(workflow)
        logger.info("[adc] [controller]: done")

    def _find_inactive_active_workflow(self, idle_ms_threshold):
        timestamp = AXWorkflow.get_current_epoch_timestamp_in_ms()
        result = {}
        with self._workflow_sets_mutex:
            for status in [AXWorkflow.RUNNING, AXWorkflow.RUNNING_DEL, AXWorkflow.RUNNING_DEL_FORCE]:
                result[status] = {}
                col = self._workflow_sets.get(status, {})
                if col:
                    for key, value in col.items():
                        diff = timestamp - value
                        if diff >= idle_ms_threshold:
                            result[status][key] = diff

        return result

    def _heartbeat_with_wfe_thread(self):
        idle_ms_threshold = 1 * 60 * 1000  # If no heartbeat for 1 minute, process overdue WFEs
        poll_interval_second = 15  # Check overdue WFEs every 15 seconds
        try:
            while True:
                time.sleep(poll_interval_second)
                all_problems = self._find_inactive_active_workflow(idle_ms_threshold=idle_ms_threshold)
                for status in [AXWorkflow.RUNNING, AXWorkflow.RUNNING_DEL, AXWorkflow.RUNNING_DEL_FORCE]:
                    problems = all_problems.get(status, {})
                    if problems:
                        if status in [AXWorkflow.RUNNING_DEL, AXWorkflow.RUNNING_DEL_FORCE]:
                            self._process_overdue_running_del(problems)
                        elif status == AXWorkflow.RUNNING:
                            self._process_overdue_running(problems)
                        else:
                            logger.critical("[adc]: bad state %s %s", status, problems)
        except Exception as e:
            logger.exception("[adc] [heartbeat]: exception")
            time.sleep(20)
            logger.error("[adc] [heartbeat] os._exit(1)")
            self.save_workflow_exception_event_to_db(exception=e)
            os._exit(1)

    def _start_heartbeat_thread(self):
        logger.info("[adc]: start heartbeat-thread")
        t = threading.Thread(name="heartbeat-thread",
                             target=self._heartbeat_with_wfe_thread,
                             kwargs={})
        t.daemon = True
        t.start()

    def _start_launcher_thread(self):
        t = threading.Thread(name="launcher", target=self._launcher_thread)
        t.daemon = True
        t.start()

    def _start_delete_resource_thread(self):
        t = threading.Thread(name="delete-resource", target=self._delete_resource_thread)
        t.daemon = True
        t.start()

    def _launcher_thread(self):
        try:
            logger.info("[adc] [launcher]: start")
            while not self._shutdown:
                with self._admitted_q_cond:
                    logger.debug("[adc] [launcher]: wait")
                    if len(self._admitted_q) == 0 and not self._shutdown:
                        self._admitted_q_cond.wait()
                    logger.debug("[adc] [launcher]: wakeup")
                    while len(self._admitted_q) and not self._shutdown:
                        workflow = self._admitted_q.pop(0)
                        self._start_worker_thread(workflow=workflow)
            logger.info("[adc] [launcher]: done")
        except Exception as e:
            logger.exception("[adc] [launcher]: exception")
            time.sleep(20)
            logger.error("[adc] [launcher] os._exit(2)")
            self.save_workflow_exception_event_to_db(exception=e)
            os._exit(2)

    def _delete_resource_thread(self):
        try:
            logger.info("[adc] [delete-resource]: start")
            while True:
                time.sleep(300)
                self._check_to_delete_resource()
                time.sleep(300)
        except Exception as e:
            logger.exception("[adc] [delete-resource]: exception")
            time.sleep(20)
            logger.error("[adc] [launcher] os._exit(2)")
            self.save_workflow_exception_event_to_db(exception=e)
            os._exit(2)

    def _check_to_delete_resource(self):
        logger.info("[adc] [delete-resource]: start to check expired resource")
        resources = AXResource.get_resources_from_db()
        for resource in resources:
            current_timestamp = AXWorkflow.get_current_epoch_timestamp_in_sec()
            if resource.timestamp + resource.ttl < current_timestamp:
                try:
                    logger.info("Found expired resource entry: %s with timestamp %s and ttl %s, but current timestamp is %s",
                                resource.resource_id, resource.timestamp, resource.ttl, current_timestamp)

                    # Call release resource function
                    self.resource_release_resource(resource.resource_id)
                    axdb_client.delete_resource(payload={'resource_id': resource.resource_id})
                    logger.info("Successfully deleted expired resource from database")
                except Exception:
                    logger.exception("[adc] [delete-resource]: failed to delete entry %s", resource)

    def _start_worker_thread(self, workflow):
        t = threading.Thread(name="worker_" + workflow.id, target=self._worker_thread_wrapper, args=(workflow,))
        t.daemon = True
        t.start()

    def _worker_thread_wrapper(self, workflow_in):
        workflow_id = workflow_in.id
        with self._worker_thread_lock:
            self._worker_thread_total += 1
            logger.info("[adc] [worker] [%s]: pre start. total work thread is %s", workflow_id, self._worker_thread_total)

        with self._revive_delete_lock:
            if workflow_id in self._revive_delete_map:
                logger.debug("[adc] [%s]: already in revive_delete_map. %s/%s, skip start",
                             workflow_id, self._revive_delete_map[workflow_id], len(self._revive_delete_map))
                do_start = False
            else:
                self._revive_delete_map[workflow_id] = 'start'
                do_start = True

        if do_start:
            self._worker_thread(workflow_in)
            with self._revive_delete_lock:
                del self._revive_delete_map[workflow_id]

        with self._worker_thread_lock:
            self._worker_thread_total -= 1
            logger.info("[adc] [worker] [%s]: post done. total work thread is %s", workflow_id, self._worker_thread_total)

    def _worker_thread(self, workflow_in):
        try:
            workflow_id = workflow_in.id
            assert workflow_in.status in [AXWorkflow.SUSPENDED, AXWorkflow.ADMITTED, AXWorkflow.ADMITTED_DEL], \
                "[adc] bad workflow {}".format(workflow_in)
            logger.info("[adc] [worker] [%s]: start", workflow_id)
            workflow = self._get_workflow_by_id_from_db(workflow_id=workflow_id)

            if workflow.status in [AXWorkflow.DELETED]:
                # no broker yet or broker not needed
                # need to release resource
                logger.info("[adc] [%s]: in DELETED", workflow_id)
                self._update_workflow_sets(workflow, None)
                self._resource_release(workflow=workflow)
                logger.info("[adc] [worker] [%s]: done (already deleted)", workflow_id)
                return
            elif workflow.status in [AXWorkflow.SUSPENDED, AXWorkflow.ADMITTED, AXWorkflow.ADMITTED_DEL]:
                might_have_broker = True
                if workflow.status in AXWorkflow.SUSPENDED:
                    # update the workflow status to ADMITTED if the status is still SUSPENDED
                    logger.info("[adc] [%s]: SUSPENDED->ADMITTED", workflow_id)
                    self._update_workflow_sets(workflow, AXWorkflow.ADMITTED)
                    if not self._update_workflow_status_in_db(workflow=workflow, new_status=AXWorkflow.ADMITTED):
                        # if failed, the new status must be DELETED
                        new_workflow = self._get_workflow_by_id_from_db(workflow_id=workflow_id)
                        assert new_workflow.status == AXWorkflow.DELETED, \
                            "[adc] bad workflow {}".format(workflow)
                        logger.info("[adc] [%s]: in DELETED", workflow_id)
                        self._update_workflow_sets(workflow, None)
                        self._resource_release(workflow=workflow)
                        logger.info("[adc] [worker] [%s]: done (just deleted)", workflow_id)
                        return
                    might_have_broker = False
                    # xxx todo: add sleep testpoint here
                    workflow = self._get_workflow_by_id_from_db(workflow_id=workflow_id)

                assert workflow.status in [AXWorkflow.ADMITTED, AXWorkflow.ADMITTED_DEL], \
                    "[adc] bad workflow {}".format(workflow)

                if workflow.status == AXWorkflow.ADMITTED_DEL:
                    self._update_workflow_sets(workflow, AXWorkflow.ADMITTED_DEL)

                # try launch workflow_executor
                if not self._launch_workflow_executor(workflow_id, max_retry=600):
                    # broker already there
                    logger.info("[adc] [%s]: workflow_executor already running", workflow_id)
                    assert might_have_broker, "[adc] bad workflow {}".format(workflow)

                # change the status to Workflow.RUNNING or Workflow.
                if workflow.status == AXWorkflow.ADMITTED:
                    logger.info("[adc] [%s]: ADMITTED->RUNNING", workflow_id)
                    new_status = AXWorkflow.RUNNING
                else:
                    logger.info("[adc] [%s]: ADMITTED_DEL->RUNNING_DEL", workflow_id)
                    new_status = AXWorkflow.RUNNING_DEL

                self._update_workflow_sets(workflow, new_status)
                if not self._update_workflow_status_in_db(workflow=workflow, new_status=new_status):
                    assert workflow.status == AXWorkflow.ADMITTED, "[adc] bad workflow {}".format(workflow)
                    workflow2 = self._get_workflow_by_id_from_db(workflow_id=workflow_id)
                    assert workflow2.status in [AXWorkflow.ADMITTED_DEL], "[adc] bad workflow {}".format(workflow)

                    self._update_workflow_sets(workflow2, AXWorkflow.RUNNING_DEL)
                    logger.info("[adc] [%s]: ADMITTED_DEL->RUNNING_DEL", workflow_id)
                    assert self._update_workflow_status_in_db(workflow=workflow2,
                                                              new_status=AXWorkflow.RUNNING_DEL), \
                        "[adc] bad workflow {}".format(workflow2)

                AXWorkflowExecutor.report_workflow_scheduled_to_kafka(workflow_id=workflow_id)
                logger.info("[adc] [worker] [%s]: done (launched)", workflow_id)
                return
            else:
                assert workflow.status in [AXWorkflow.RUNNING, AXWorkflow.RUNNING_DEL, AXWorkflow.RUNNING_DEL_FORCE, AXWorkflow.FAILED,
                                           AXWorkflow.FORCED_FAILED, AXWorkflow.SUCCEED], \
                    "[adc] bad workflow {}".format(workflow)
                # this should never happen
                assert False, "[adc] bad workflow {}".format(workflow)

            assert False, "[adc] [{}] should never reached here".format(workflow_id)

        except Exception as e:
            logger.exception("[adc] [worker] exception")
            time.sleep(20)
            logger.error("[adc] [worker] os._exit(3)")
            self.save_workflow_exception_event_to_db(exception=e)
            os._exit(3)

    def _add_suspended_workflow_to_q(self, workflow):
        assert workflow.status == AXWorkflow.SUSPENDED, "[adc] bad workflow {}".format(workflow)
        with self._suspended_q_cond:
            logger.info("[adc] [controller] [%s]: add to suspended Q", workflow.id)
            self._update_workflow_sets(workflow, AXWorkflow.SUSPENDED)
            self._suspended_q.append(workflow)
            self._suspended_map[workflow.id] = workflow
            self._suspended_q_cond.notify_all()

    def _add_admitted_workflow_to_q(self, workflow):
        with self._admitted_q_cond:
            logger.info("[adc] [controller] [%s]: add to admitted Q ", workflow.id)
            self._admitted_q.append(workflow)
            self._admitted_q_cond.notify_all()

    def _resource_reserve(self, workflow):
        # assert self._suspended_q_cond.locked()
        if workflow.id not in self._resource_reserving_workflow_set.keys():
            self._used_resource += workflow.resource
            self._used_resource += self._executor_resource
            self._resource_reserving_workflow_set[workflow.id] = workflow.resource.resource
            logger.info("[adc] [resource] [%s]: reserve resource %s(%s/%s)",
                        workflow.id, workflow.resource,
                        self._used_resource, self._total_resource)
        else:
            logger.info("[adc] [resource] [%s]: NOT reserve resource %s(%s/%s)",
                        workflow.id, workflow.resource,
                        self._used_resource, self._total_resource)

    def _resource_release(self, workflow):
        # assert self._suspended_q_cond.locked()
        if workflow.id in self._resource_reserving_workflow_set.keys():
            self._used_resource -= AXWorkflowResource(self._resource_reserving_workflow_set[workflow.id])
            self._used_resource -= self._executor_resource
            self._resource_reserving_workflow_set.pop(workflow.id)
            logger.info("[adc] [resource] [%s]: release resource %s(%s/%s)",
                        workflow.id, workflow.resource,
                        self._used_resource, self._total_resource)
            return True
        else:
            logger.info("[adc] [resource] [%s]: NOT release resource %s(%s/%s)",
                        workflow.id, workflow.resource,
                        self._used_resource, self._total_resource)
            return False

    def _resource_update(self, workflow_id, new_resource):
        # assert self._suspened_q_cond.locked()
        if workflow_id in self._resource_reserving_workflow_set.keys():
            old_resource = AXWorkflowResource(self._resource_reserving_workflow_set[workflow_id])

            logger.info("new %s and old %s", new_resource, old_resource)
            if new_resource < old_resource:
                resource_delta = new_resource - old_resource
                self._used_resource += resource_delta
                self._resource_reserving_workflow_set[workflow_id] = new_resource.resource
                logger.info("[adc] [resource] [%s]: update resource from %s to %s (%s/%s)",
                            workflow_id, old_resource, new_resource,
                            self._used_resource, self._total_resource)
                return True
            else:
                logger.info("[adc] [resource] [%s]: NOT update resource %s(%s/%s)",
                            workflow_id, old_resource, self._used_resource, self._total_resource)
        return False

    def check_to_reserve(self, resource):
        assert isinstance(resource, AXWorkflowResource), "parameter must be resource type"
        if resource + self._used_resource <= self._total_resource:
            return True
        return False

    def resource_reserve_resource(self, resource_json):
        """
        Reserve resource for a resource
        :param resource_json:
        :return:
        """
        logger.info("[adc] [resource] start reserving resource for %s", resource_json)
        resource_obj = AXResource.get_resource_from_payload(resource_json)
        resource_obj_id = resource_obj.resource_id
        prev_resource_obj = None

        with self._suspended_q_cond:
            if resource_obj_id in self._resource_reserving_intermediate_set:
                err_msg = "[adc] [resource] [{}]: pending state".format(resource_obj_id)
                logger.error(err_msg)
                raise AXIllegalOperationException(err_msg)

            if resource_obj_id not in self._resource_reserving_resource_set:
                if self.check_to_reserve(resource_obj.resource):
                    self._used_resource += resource_obj.resource
                    self._resource_reserving_intermediate_set.add(resource_obj_id)
                else:
                    err_msg = "[adc] [resource] [{}]: not enough resource for resource {}({}/{})". \
                        format(resource_obj_id, resource_obj.resource, self._used_resource, self._total_resource)
                    logger.error(err_msg)
                    raise AXIllegalOperationException(err_msg)
            else:
                prev_resource_obj = self._resource_reserving_resource_set[resource_obj_id]
                delta_resource = resource_obj.resource - prev_resource_obj.resource
                logger.info("[adc] [resource] updating reserved resource, delta: %s", delta_resource)

                if self.check_to_reserve(delta_resource):
                    self._used_resource += delta_resource
                    self._resource_reserving_intermediate_set.add(resource_obj_id)
                else:
                    err_msg = "[adc] [resource] [{}]: not enough resource for updating resource {}({}/{})". \
                        format(resource_obj_id, resource_obj.resource, self._used_resource, self._total_resource)
                    logger.error(err_msg)
                    raise AXIllegalOperationException(err_msg)

        if prev_resource_obj is None:
            current_timestamp = AXWorkflow.get_current_epoch_timestamp_in_sec()
            try:
                axdb_client.create_resource(payload={'resource_id': resource_obj_id,
                                                     'category': resource_obj.category,
                                                     'resource': json.dumps(resource_obj.resource.resource),
                                                     'ttl': resource_obj.ttl,
                                                     'timestamp': current_timestamp,
                                                     'detail': resource_obj.detail})
            except Exception:
                # Failed to update db, revert changes in the memory
                with self._suspended_q_cond:
                    self._used_resource -= resource_obj.resource
                    self._resource_reserving_resource_set.pop(resource_obj_id, None)
                    self._resource_reserving_intermediate_set.remove(resource_obj_id)
                    err_msg = "[adc] [resource] [{}]: fail to create resource to db".format(resource_obj_id)
                    logger.error(err_msg)
                    raise AXIllegalOperationException(err_msg)
            with self._suspended_q_cond:
                resource_obj.timestamp = current_timestamp
                self._resource_reserving_resource_set[resource_obj_id] = resource_obj
                self._resource_reserving_intermediate_set.remove(resource_obj_id)
                return {}
        else:
            current_timestamp = AXWorkflow.get_current_epoch_timestamp_in_sec()
            try:
                axdb_client.update_resource_conditionally(payload={'resource_id': resource_obj_id,
                                                                   'category': resource_obj.category,
                                                                   'resource': json.dumps(resource_obj.resource.resource),
                                                                   'ttl': resource_obj.ttl,
                                                                   'detail': resource_obj.detail,
                                                                   'timestamp': current_timestamp,
                                                                   'timestamp_update_if': prev_resource_obj.timestamp})
            except Exception:
                # Failed to update db, revert changes in the memory
                with self._suspended_q_cond:
                    delta_resource = resource_obj.resource - prev_resource_obj.resource
                    self._used_resource -= delta_resource
                    self._resource_reserving_intermediate_set.remove(resource_obj_id)
                    err_msg = "[adc] [resource] [{}]: fail to update resource to db".format(resource_obj_id)
                    logger.error(err_msg)
                    raise AXIllegalOperationException(err_msg)
            with self._suspended_q_cond:
                resource_obj.timestamp = current_timestamp
                self._resource_reserving_resource_set[resource_obj_id] = resource_obj
                self._resource_reserving_intermediate_set.remove(resource_obj_id)
                return {}

    def resource_release_resource(self, resource_id):
        """
        Release resource for a resource
        :param resource_id:
        :return:
        """
        logger.info("[adc] [resource] start releasing resource for %s", resource_id)

        if resource_id is None:
            err_msg = "[adc] [resource]: resource release failed due to id is None"
            logger.info(err_msg)
            raise AXIllegalArgumentException(err_msg)

        prev_resource_obj = None
        with self._suspended_q_cond:
            if resource_id in self._resource_reserving_intermediate_set:
                err_msg = "[adc] [resource] [{}]: pending state".format(resource_id)
                logger.error(err_msg)
                raise AXIllegalOperationException(err_msg)

            if resource_id in self._resource_reserving_resource_set:
                prev_resource_obj = self._resource_reserving_resource_set[resource_id]
                self._used_resource -= prev_resource_obj.resource
                self._resource_reserving_intermediate_set.add(resource_id)
            else:
                err_msg = "[adc] [resource] [{}]: cannot find resource.".format(resource_id)
                logger.info(err_msg)
                return {}

        try:
            # TODO: Use conditional delete if available (AA-2398)
            axdb_client.delete_resource(payload={'resource_id': resource_id})
        except Exception:
            # Failed to update db, revert changes in the memory
            with self._suspended_q_cond:
                self._used_resource += prev_resource_obj.resource
                self._resource_reserving_intermediate_set.remove(resource_id)
                err_msg = "[adc] [resource] [{}]: fail to release resource to db".format(resource_id)
                logger.error(err_msg)
                raise AXIllegalOperationException(err_msg)
        with self._suspended_q_cond:
            self._resource_reserving_resource_set.pop(resource_id, None)
            self._resource_reserving_intermediate_set.remove(resource_id)
            return {}

    def _prepare_executor_service_template(self, workflow_id, costid=None):
        # use workflow_id as part of the executor's name
        service_template = copy.deepcopy(self._service_template)
        service_template["id"] = workflow_id
        service_template["service_context"] = {"parent_service_instance_id": workflow_id,
                                               "name": service_template["template"]["name"],
                                               "service_instance_id": workflow_id,
                                               "root_workflow_id": workflow_id}
        if costid:
            service_template["costid"] = costid

        container_name = AxsysClient.guess_container_full_name(service_template=service_template, expand_override=True)
        # to make workflow-executor, which is marked as long lived container (so it can be restart), unique
        service_template["name"] = container_name

        if container_name is None:
            assert False, "[adc] [{}]: invalid workflow executor {}".format(workflow_id,
                                                                            pprint.pformat(service_template))
        return container_name, service_template

    def _launch_workflow_executor(self, workflow_id, max_retry=600):
        w = self._get_workflow_by_id_from_db(workflow_id=workflow_id, need_load_template=True)
        costid = w.service_template.get("costid", None)
        del w
        container_name, service_template = self._prepare_executor_service_template(workflow_id=workflow_id, costid=costid)

        cmd = "/ax/src/ax/devops/workflow/workflowexecutor"
        url = "http://axworkflowadc.axsys:{port}/v1/adc/notification/workflow".format(port=self._port)
        command = "python3 {cmd} --self-container-name {container_name} --workflow-id {workflow_id} --report-done-url {url} " \
                  "--ax-sys-cpu-core {cpu} --ax-sys-mem-mib {mem} --vol-size {vol_size} --instance-type {instance_type}".\
            format(cmd=cmd, container_name=container_name, workflow_id=workflow_id, url=url, cpu=self.instance_sys_resource[0],
                   mem=self.instance_sys_resource[1], vol_size=self.max_vol_size, instance_type=self.instance_type)
        service_template["template"]["command"] = command.split(" ")
        service_template["template"]["args"] = []
        image = service_template["template"]["image"]
        image = image.replace("%%registry%%", self._wfe_registry)
        image = image.replace("%%name_space%%", self._wfe_namespace)
        image = image.replace("%%version%%", self._wfe_version)
        service_template["template"]["image"] = image

        logger.info("[adc] [%s]: launch ax_workflow_executor, command=%s, container_name=%s",
                    workflow_id, command, container_name)

        AXWorkflow.service_template_add_reporting_callback_param(service_template=service_template,
                                                                 instance_id=workflow_id,
                                                                 auto_retry=True,
                                                                 is_wfe=True)
        counter = 0
        while counter < max_retry:
            counter += 1
            logger.info("[adc] [%s]: about to launch container %s", workflow_id, container_name)
            rc, containers, response_status_code, response_json = axsys_client.create_service(service_template=service_template)
            if rc:
                remote_container_name = axsys_client.canonical_container_full_name(containers[0])
                logger.info("[adc] [%s]: container %s launched", workflow_id, remote_container_name)
                assert remote_container_name == container_name, "name mismatch {} vs {}".format(remote_container_name,
                                                                                                container_name)
                return True
            else:
                logger.info("[adc] [%s]: cannot launch container %s, response_status_code=%s, response_json=%s",
                            workflow_id, container_name, response_status_code, response_json)
                container_status = axsys_client.get_container_status(container_name=container_name)
                if container_status in [AxsysClient.CONTAINER_RUNNING, axsys_client.CONTAINER_IMAGE_PULL_BACKOFF, axsys_client.CONTAINER_PENDING]:
                    logger.info("[adc] [%s]: container %s is still %s", workflow_id, container_name, container_status)
                    return False
                elif container_status in [AxsysClient.CONTAINER_STOPPED, AxsysClient.CONTAINER_FAILED]:
                    # delete it
                    logger.info("[adc] [%s]: delete stopped %s",
                                workflow_id, container_name)
                    rc_del, result_del = axsys_client.delete_service(service_name=container_name)
                    logger.info("[adc] [%s] delete %s return %s %s",
                                workflow_id, container_name, rc_del, result_del)
                else:
                    logger.info("[adc] [%s]: %s in %s",
                                workflow_id, container_name, container_status)
                sleep_time = 60
                logger.info("[adc] [%s]: sleep %s seconds and retry %s",
                            workflow_id, sleep_time, counter)
                time.sleep(sleep_time)
                logger.info("[adc] [%s]: slept %s seconds",
                            workflow_id, sleep_time)
                self._refresh_workflow_sets_by_id(workflow_id)

        assert False, "[adc] [{}]: cannot launch workflow executor {}".format(workflow_id, pprint.pformat(service_template))

    def _delete_container_running_del_thread(self, workflow_id, max_retry):
        try:
            with self._revive_delete_lock:
                if workflow_id in self._revive_delete_map:
                    logger.debug("[adc] [%s]: already in revive_delete_map. %s/%s, skip delete",
                                 workflow_id, self._revive_delete_map[workflow_id], len(self._revive_delete_map))
                    return
                else:
                    self._revive_delete_map[workflow_id] = 'delete'
            self._delete_container_running_del(workflow_id=workflow_id, max_retry=max_retry)
        except Exception:
            logger.exception("[adc] [%s]: exeception in force deletion", workflow_id)

        with self._revive_delete_lock:
            del self._revive_delete_map[workflow_id]

    def _delete_container_running_del(self, workflow_id, force=False, max_retry=6):
        container_name, service_template = self._prepare_executor_service_template(workflow_id=workflow_id)
        count = 0
        while True:
            container_status = axsys_client.get_container_status(container_name=container_name)
            if container_status in [AxsysClient.CONTAINER_RUNNING, axsys_client.CONTAINER_IMAGE_PULL_BACKOFF, axsys_client.CONTAINER_PENDING]:
                logger.info("[adc] [%s]: container %s is still %s",
                            workflow_id, container_name, container_status)
                return
            else:
                if container_status in [AxsysClient.CONTAINER_STOPPED, AxsysClient.CONTAINER_FAILED]:
                    # delete it
                    logger.info("[adc] [%s]: delete stopped %s", workflow_id, container_name)
                    rc_del, result_del = axsys_client.delete_service(service_name=container_name)
                    logger.info("[adc] [%s] delete %s return %s %s",
                                workflow_id, container_name, rc_del, result_del)
                else:
                    logger.info("[adc] [%s]: %s in %s",
                                workflow_id, container_name, container_status)

                workflow = self._get_workflow_by_id_from_db(workflow_id=workflow_id)
                if workflow and workflow.status in [AXWorkflow.RUNNING_DEL, AXWorkflow.RUNNING_DEL_FORCE]:
                    # no executor container and still in RUNNING_DEL state
                    count += 1
                    if count <= max_retry:
                        logger.info("[adc] [%s]: in %s state but no executor running, retry %s/%s",
                                    workflow_id, workflow.status, count, max_retry)
                        time.sleep(10)
                        continue
                    else:
                        logger.warning("[adc] [%s]: force update to %s",
                                       workflow_id, AXWorkflow.DELETED)
                        self._update_workflow_sets(workflow, None)
                        if not self._update_workflow_status_in_db(workflow=workflow, new_status=AXWorkflow.DELETED):
                            workflow2 = self._get_workflow_by_id_from_db(workflow_id=workflow_id)
                            assert workflow2.status == AXWorkflow.DELETED, "[adc] [{}] bad status {}".format(workflow_id, workflow2)
                        else:
                            logger.warning("[adc] [%s]: force update to %s done",
                                           workflow_id, AXWorkflow.DELETED)
                            AXWorkflowExecutor.report_workflow_force_termination_to_kafka(workflow_id=workflow_id)
                        AXWorkflowEvent.save_workflow_event_to_db(workflow_id=workflow_id,
                                                                  event_type=AXWorkflowEvent.FORCE_DELETE)
                        return
                else:
                    logger.warning("[adc] [%s]: status=%s ", workflow_id, workflow.status)
                    return

    def _process_overdue_running_del(self, problems):
        for workflow_id, overdue in problems.items():
            # to avoid multiple action for the same workflow at the same time
            self._refresh_workflow_sets_by_id(workflow_id)
            thread_name = "force-delete-thread-{}".format(workflow_id)
            logger.info("[adc] [%s]: start %s to delete overdue workflow in running_del. overdue=%s",
                        workflow_id, thread_name, overdue)
            t = threading.Thread(name=thread_name,
                                 target=self._delete_container_running_del_thread,
                                 kwargs={'workflow_id': workflow_id,
                                         'max_retry': 12})
            t.daemon = True
            t.start()

    def _process_overdue_running(self, problems):
        for workflow_id, overdue in problems.items():
            # to avoid multiple action for the same workflow at the same time
            # self._refresh_workflow_sets_by_id(workflow_id)
            thread_name = "force-revive-thread-{}".format(workflow_id)
            logger.info("[adc] [%s]: start %s to revive overdue workflow in running. overdue=%s",
                        workflow_id, thread_name, overdue)
            t = threading.Thread(name=thread_name,
                                 target=self._revive_container_running_thread,
                                 kwargs={'workflow_id': workflow_id,
                                         'max_retry': 2,
                                         'overdue': overdue})
            t.daemon = True
            t.start()

    def _revive_container_running_thread(self, workflow_id, max_retry, overdue):
        try:
            with self._revive_delete_lock:
                if workflow_id in self._revive_delete_map:
                    logger.debug("[adc] [%s]: already in revive_delete_map. %s/%s, skip revive",
                                 workflow_id, self._revive_delete_map[workflow_id], len(self._revive_delete_map))
                    return
                else:
                    self._revive_delete_map[workflow_id] = 'revive'
            self._revive_container_running(workflow_id=workflow_id, max_retry=max_retry, overdue=overdue)
        except Exception:
            logger.exception("[adc] [%s]: exception in force revive", workflow_id)

        with self._revive_delete_lock:
            del self._revive_delete_map[workflow_id]

    def _revive_container_running(self, workflow_id, max_retry, overdue):
        if not self._launch_workflow_executor(workflow_id=workflow_id, max_retry=max_retry):
            logger.info("[adc] [%s]: check workflow executor overdue %s", workflow_id, overdue)

            if overdue > 20*60*1000:  # AA-1813 no heart beat from wfe but container is still in RUNNING statue for 20 minutes
                container_name, service_template = self._prepare_executor_service_template(workflow_id=workflow_id)
                err_msg = "[adc] [{}]: delete workflow executor due to hanging for long time without heartbeat {}".format(workflow_id, container_name)
                logger.info(err_msg)

                # We want to send an alert message to notification center here.
                # However, we cannot flood the notification center with this alert messages since this reporting happens every minute.
                send_notification = False
                with self._notification_center_lock:
                    if workflow_id not in self._notification_center_list:
                        self._notification_center_list[workflow_id] = AXWorkflow.get_current_epoch_timestamp_in_ms()
                        send_notification = True

                    for k, v in self._notification_center_list.items():
                        if AXWorkflow.get_current_epoch_timestamp_in_ms() - v > 24 * 60 * 60 * 1000:
                            # The workflow has been in the notification list for more than a day, remove it in order to send notification
                            logger.info("[adc] [{}] delete workflow from notification center list for over a day", workflow_id)
                            del self._notification_center_list[k]

                if send_notification:
                    try:
                        event_notification_client.send_message_to_notification_center(
                            CODE_ADC_MISSING_HEARTBEAT_FROM_WFE,
                            detail={'message': "[adc] [{}] Have not received heartbeat from workflow executor {}, please contact Argo support team.".format(workflow_id, container_name)})
                    except Exception:
                        logger.exception("Failed to send event to notification center")

                return  # xxx TODO: need to be remove!!! see AA-2359

                rc_del, result_del = axsys_client.delete_service(service_name=container_name)
                logger.info("[adc] [%s] delete %s return %s %s", workflow_id, container_name, rc_del, result_del)
                self._launch_workflow_executor(workflow_id=workflow_id, max_retry=max_retry)
                logger.info("[adc] [%s] relaunching workflow executor after deleting the hanging one.", workflow_id)
            else:
                return
        else:
            self._refresh_workflow_sets_by_id(workflow_id)

        AXWorkflowEvent.save_workflow_event_to_db(workflow_id=workflow_id, event_type=AXWorkflowEvent.FORCE_START)

    @staticmethod
    def _post_workflow_to_db(workflow):
        return AXWorkflow.post_workflow_to_db(workflow)

    @staticmethod
    def _get_workflow_by_id_from_db(workflow_id, need_load_template=False):
        return AXWorkflow.get_workflow_by_id_from_db(workflow_id=workflow_id, need_load_template=need_load_template)

    @staticmethod
    def _get_workflows_by_status_from_db(status):
        ret = AXWorkflow.get_workflows_by_status_from_db(status=status)
        logger.info("adc load %s workflow whose status=%s", len(ret), status)
        return ret

    @staticmethod
    def _get_resources_from_db():
        ret = AXResource.get_resources_from_db()
        logger.info("adc load %s resources", len(ret))
        return ret

    @staticmethod
    def _update_workflow_status_in_db(workflow, new_status):
        return AXWorkflow.update_workflow_status_in_db(workflow, new_status)

    def workflow_create_random(self, input_json):
        crash_second = (input_json and input_json.get("crash_second", 0)) or 0
        base_image = "workflow"
        if not input_json:
            input_json = {}
        input_json["image"] = "{}/{}/{}:{}".format(self._wfe_registry, self._wfe_namespace,
                                                   base_image, self._wfe_version)
        workflow_json, node_count, failure_node, leaf_node = gen_random_workflow(parameter=input_json,
                                                                                 service_id=str(uuid.uuid4()),
                                                                                 name_prefix="axrandom",
                                                                                 depth=1)
        logger.info("[adc]: Creating random workflow with %s/%s/%s node. param=%s crash_second=%s",
                    node_count, failure_node, leaf_node, input_json, crash_second)
        test_tags = {AXWorkflow.tag_test_ax_workflow_executor_crash_second: crash_second,
                     AXWorkflow.tag_test_ax_workflow_expect_failure_leaf_node: failure_node}
        workflow_json[AXWorkflow.tag_test_ax_workflow] = test_tags
        ret = self.workflow_create(workflow_json)
        ret["node_count"] = node_count
        ret["leaf_node"] = leaf_node
        if failure_node:
            ret["failure_node_count"] = failure_node
        if crash_second:
            ret["crash_second"] = crash_second
        return ret

    def workflow_create(self, workflow_json):
        logger.info("[adc]: Creating workflow ...")
        workflow_id = workflow_json.get("id", None)
        if workflow_id is None:
            logger.info("[adc]: Rejected workflow:\n%s", pprint.pformat(workflow_json))
            raise AXIllegalArgumentException("[adc]: Workflow does't have id. {}".format(pprint.pformat(workflow_json)))
        if self.state not in [ADCState.RUNNING, ADCState.SUSPENDED_ALLOW_NEW]:
            logger.info("[adc] [%s]: state=%s Rejected workflow:\n", workflow_id, self.state)
            msg = "[adc]: Workflow creations forbidden in {} state".format(self.state)
            if self.state in [ADCState.UNKNOWN, ADCState.STARTING, ADCState.STOPPED]:
                raise AXServiceTemporarilyUnavailableException(msg)
            else:
                raise AXIllegalOperationException(msg)
        try:
            logger.info('[adc] [%s]: substituting parameters', workflow_id)
            substitute_parameters(workflow_json)
            logger.info("[adc] [%s]: workflow:\n%s", workflow_id, pprint.pformat(workflow_json))
        except Exception:
            logger.exception("[adc]: bad workflow %s", pprint.pformat(workflow_json))
            raise AXIllegalArgumentException("[adc]: bad workflow {}".format(str(traceback.format_exc())))

        # Calculate resource
        resource_payload = {
            'ax_cpu_core': self.instance_sys_resource[0],
            'ax_mem_mib': self.instance_sys_resource[1],
            'max_vol_size': self.max_vol_size,
            'instance_type': self.instance_type,
        }
        root_ndoe = AXWorkflowExecutor.build_tree(workflow_id=workflow_id, payload=workflow_json, name="root", full_path='',
                                                  parent_node=None, is_fixture=False, leaf_nodes=dict(), executor=None,
                                                  resource_payload=resource_payload, dry_run=True)

        resource = root_ndoe.max_resource
        leaf_resource = root_ndoe.max_leaf_resource
        self._check_resource(resource, leaf_resource)

        workflow = AXWorkflow(workflow_id=workflow_id, service_template=workflow_json,
                              timestamp=AXWorkflow.get_current_epoch_timestamp_in_ms(),
                              resource=resource.resource, leaf_resource=leaf_resource.resource)

        # lookup suspended_map first
        w = None
        with self._suspended_q_cond:
            if workflow.id in self._suspended_map:
                logger.info("[adc] [%s]: possible duplicated request. (same workflow_id found in suspended_map, did not check service_template)", workflow_id)
                return {}

        # try to save to DB if not in suspended_map
        if w is None and not self._post_workflow_to_db(workflow=workflow):
            w = self._get_workflow_by_id_from_db(workflow_id=workflow_id, need_load_template=True)

        if not w is None:
            if w == workflow:
                # existing workflow
                logger.info("[adc] [%s]: duplicated request", workflow_id)
                return {}
            else:
                logger.error("[adc] [%s]: different workflow with same id is already in DB, reject", workflow_id)
                raise AXIllegalArgumentException("[adc]: There is already a different workflow {} with same workflow id. {}"
                                                 .format(pprint.pformat(w.service_template), pprint.pformat(workflow_json)))

        # add to suspended_workflow_q
        workflow.free_template()  # to save memory
        self._add_suspended_workflow_to_q(workflow)
        logger.info("[adc] [%s]: create request processed", workflow_id)
        return {"id": workflow_id}

    def _check_resource(self, resource, leaf_resource):
        if not self._total_resource >= resource + self._executor_resource:
            msg = "AWS cluster's total resource cannot accommodate the workflow."
            raise AXIllegalArgumentException(msg)

        if not self._instance_resource >= leaf_resource:
            msg = "AWS instance's total resource cannot accommodate one of the containers in the workflow."
            raise AXIllegalArgumentException(msg)

    def _workflow_delete_no_exception(self, workflow_id):
        try:
            return self.workflow_delete(workflow_id)
        except Exception:
            logger.exception("")
            return {}

    def workflow_delete(self, workflow_id, force=False):
        if workflow_id is None:
            logger.error("[adc]: Reject workflow deletion request without id")
            raise AXIllegalArgumentException("[adc]: Workflow deletion request does't have id.")

        logger.info("[adc] [%s]: delete request force=%s", workflow_id, force)
        while True:
            workflow = self._get_workflow_by_id_from_db(workflow_id=workflow_id)
            if workflow:
                current_status = workflow.status
                logger.info("[adc] [%s]: status=%s", workflow_id, current_status)
                if force and current_status not in [AXWorkflow.RUNNING_DEL, AXWorkflow.RUNNING_DEL_FORCE]:
                    raise AXIllegalArgumentException("[adc] [%s]: can only force delete a workflow which is in RUNNING_DEL state")

                if current_status == AXWorkflow.SUSPENDED:
                    new_status = AXWorkflow.DELETED
                elif current_status == AXWorkflow.ADMITTED:
                    new_status = AXWorkflow.ADMITTED_DEL
                elif current_status == AXWorkflow.RUNNING:
                    new_status = AXWorkflow.RUNNING_DEL
                elif current_status in [AXWorkflow.ADMITTED_DEL, AXWorkflow.DELETED]:
                    logger.info("[adc] [%s]: Workflow already in %s status", workflow_id, current_status)
                    return {}
                elif current_status in [AXWorkflow.RUNNING_DEL]:
                    if not force:
                        logger.info("[adc] [%s]: Workflow already in %s status", workflow_id, current_status)
                        return {}
                    else:
                        new_status = AXWorkflow.RUNNING_DEL_FORCE
                elif current_status in AXWorkflow.RUNNING_DEL_FORCE:
                    if not force:
                        logger.info("[adc] [%s]: Workflow already in %s status", workflow_id, current_status)
                    else:
                        self._delete_container_running_del(workflow_id=workflow.id)
                    return {}
                elif current_status == AXWorkflow.FAILED or current_status == AXWorkflow.FORCED_FAILED:
                    logger.info("[adc] [%s]: Workflow already failed.", workflow_id)
                    self._update_workflow_sets(workflow, None)
                    raise AXWorkflowAlreadyFailed("[adc] [{}]: Workflow already failed.".format(workflow_id))
                elif current_status == AXWorkflow.SUCCEED:
                    logger.info("[adc] [%s]: Workflow already succeed.", workflow_id)
                    self._update_workflow_sets(workflow, None)
                    raise AXWorkflowAlreadySucceed("[adc] [{}]: Workflow already succeed.".format(workflow_id))
                else:
                    assert False, "[adc] bad workflow {}".format(workflow)

                logger.info("[adc] [%s]: %s->%s", workflow_id, current_status, new_status)
                if new_status in [AXWorkflow.DELETED]:
                    self._update_workflow_sets(workflow, None)
                else:
                    self._update_workflow_sets(workflow, new_status)
                if self._update_workflow_status_in_db(workflow=workflow,
                                                      new_status=new_status):
                    if new_status == AXWorkflow.DELETED:
                        AXWorkflowExecutor.report_workflow_cancel_to_kafka(workflow_id=workflow_id)

                    if new_status == AXWorkflow.RUNNING_DEL_FORCE:
                        key = AXWorkflow.REDIS_DEL_FORCE_LIST_KEY
                    else:
                        key = AXWorkflow.REDIS_DEL_LIST_KEY
                    redis_client.rpush(key=key.format(workflow_id),
                                       value={"status": new_status},
                                       expire=AXWorkflow.REDIS_LIST_EXPIRE_SECONDS,
                                       encoder=json.dumps)
                    logger.info("[adc] [%s]: delete request processed", workflow_id)
                    return {}
                else:
                    logger.info("[adc] [%s]: retry update", workflow_id)
                    continue
            else:
                raise AXWorkflowDoesNotExist("[adc] [{}]: Workflow does't exist.".format(workflow_id))

    def workflow_show(self, workflow_id, state_only):
        if workflow_id is None:
            logger.error("[adc]: Reject workflow show request without id")
            raise AXIllegalArgumentException("[adc]: Workflow show request does't have id.")

        logger.info("[adc] [%s]: show request", workflow_id)

        workflow_events = [i.jsonify() for i in AXWorkflowEvent.load_events_from_db(workflow_id=workflow_id)]
        if workflow_id == ADC_WORKFLOW_ID:
            return {"events": workflow_events}

        workflow = self._get_workflow_by_id_from_db(workflow_id=workflow_id, need_load_template=True)
        ret = None
        if workflow:
            logger.info("[adc] [%s]: status=%s resource=%s", workflow_id, workflow.status, workflow.resource)

            ret = {"workflow_id": workflow_id,
                   "status": workflow.status,
                   "resource": workflow.resource.resource,
                   "leaf_resource": workflow.leaf_resource.resource,
                   "timestamp": workflow.timestamp}
            if not state_only:
                ret["service_template"] = workflow.service_template
                if workflow_events:
                    workflow_events.reverse()
                    ret["events"] = workflow_events
                try:
                    leaf_results = AXWorkflowNodeResult.load_results_from_db(workflow_id)
                    results = []
                    for result in leaf_results:
                        results.append({"sn": result.sn,
                                        "leaf_id": result.node_id,
                                        "result_code": result.result_code,
                                        "detail": result.detail,
                                        "timestamp": result.timestamp})

                    results.reverse()
                    ret["leaf_results"] = results
                except Exception:
                    logger.exception("[adc] [%s]: cannot get leaf_results", workflow_id)

                # get axworkflowexecutor result
                container_result, _ = AXWorkflow.get_workflow_leaf_result(workflow_id)
                if container_result:
                    try:
                        container_result = json.loads(container_result)
                        if container_result["event_type"] == "HAVE_RESULT":
                            ret["axworkflowexecutor_callback_result"] = container_result
                    except Exception:
                        logger.exception("[adc] [%s] cannot decode result", workflow_id)

        try:
            leaf_results = AXWorkflowNodeResult.get_leaf_service_results_by_leaf_id_from_db(leaf_id=workflow_id)
            if leaf_results:
                if ret is None:
                    ret = {}
                ret.update({
                    "workflow_id": leaf_results[0].workflow_id,
                    "leaf_id": workflow_id
                })
                results = []
                latest = None
                for result in leaf_results:
                    r = {"sn": result.sn,
                         "result_code": result.result_code,
                         "detail": result.detail,
                         "timestamp": result.timestamp}
                    results.append(r)
                    if not latest or latest.sn < result.sn:
                        latest = result
                results.reverse()
                if not state_only:
                    ret["leaf_results"] = results

                state = WorkflowNode.UNKNOWN_STATE
                if latest:
                    if latest.result_code == AXWorkflowNodeResult.LAUNCHED:
                        state = WorkflowNode.LAUNCHED_STATE
                    elif latest.result_code == AXWorkflowNodeResult.SUCCEED:
                        state = WorkflowNode.SUCCEED_STATE
                    elif latest.result_code == AXWorkflowNodeResult.FAILED:
                        state = WorkflowNode.FAILED_STATE
                    elif latest.result_code == AXWorkflowNodeResult.INTERRUPTED:
                        state = WorkflowNode.INTERRUPTED_STATE
                ret["state"] = state
        except Exception:
            logger.exception("[adc] [%s]: cannot get leaf_results", workflow_id)

        if ret is not None:
            logger.info("[adc] [%s]: show workflow done", workflow_id)
            return ret
        raise AXIllegalArgumentException("[adc] [{}]: Workflow does't exist.".format(workflow_id))

    def notification_workflow(self, workflow_json):
        workflow_id = workflow_json.get("workflow_id", None)
        new_status = workflow_json.get("last_status", None)
        event = workflow_json.get("event", False)

        if workflow_id is None:
            logger.info("[adc]: Reject workflow notification without id:\n%s", pprint.pformat(workflow_json))
            raise AXIllegalArgumentException("[adc]: Workflow does't have id. {}".format(pprint.pformat(workflow_json)))

        if self.state not in [ADCState.RUNNING, ADCState.SUSPENDED_ALLOW_NEW, ADCState.SUSPENDED_NO_NEW,
                              ADCState.STOPPED]:
            # the caller should retry
            logger.info("[adc] [resource] [%s]: reject notification because ADC is not ready", workflow_id)
            raise AXServiceTemporarilyUnavailableException("[adc] [{}]: ADC is not ready".format(workflow_id))

        if event == "done":
            logger.info("[adc] [resource] [%s]: notification workflow_status=%s detail=%s",
                        workflow_id, new_status, workflow_json)
            self._update_workflow_sets_by_id(workflow_id, None)
            workflow = self._get_workflow_by_id_from_db(workflow_id=workflow_id)
            if workflow:
                with self._suspended_q_cond:
                    if self._resource_release(workflow):
                        self._suspended_q_cond.notify_all()

                    logger.info("[adc] [resource] [%s]: notification request processed.",
                                workflow)
                    return {}
            else:
                raise AXIllegalArgumentException("[adc] [{}]: Workflow does't exist.".format(workflow_id))
        else:
            logger.debug("[adc] [%s]: heartbeat", workflow_id)
            self._refresh_workflow_sets_by_id(workflow_id)

            if event == "workflow_info":
                logger.info("[adc] [%s]: got workflow_info.", workflow_id)
                nodes = workflow_json.get("nodes", {})
                output = self._workflows_info.get(workflow_id, {})
                output["nodes"] = nodes

            workflow_resource = None
            try:
                workflow_resource = AXWorkflowResource(json.loads(workflow_json.get("resource")))
            except Exception:
                logger.exception("Failed to parse resource")

            if event == "heartbeat" and workflow_resource:
                with self._suspended_q_cond:
                    if self._resource_update(workflow_id, workflow_resource):
                        self._suspended_q_cond.notify_all()
                        logger.info("[adc] [resource] [%s]: notification request heartbeat caused resource update.",
                                    workflow_id)

            return {}

    def notification_resource(self, resource_json):
        # XXX todo: check max per-node resource, platform
        total_resource = resource_json.get("total_resource", None)

        if total_resource and isinstance(total_resource, numbers.Number):
            with self._suspended_q_cond:
                logger.info("[adc] [resource]: update resource from %s to %s, used=%s",
                            self._total_resource, total_resource, self._used_resource)
                self._total_resource = total_resource
                self._suspended_q_cond.notify_all()
                logger.info("[adc] [resource]: notification request processed")
                return {}
        else:
            raise AXIllegalArgumentException("[adc] [resource]: no total_resource in {}".format(resource_json))

    @staticmethod
    def save_workflow_exception_event_to_db(exception):
        AXWorkflowEvent.save_workflow_exception_event_to_db(workflow_id=ADC_WORKFLOW_ID, exception=exception)

    @staticmethod
    def save_workflow_start_event_to_db():
        counter = 0
        max_retry = 180
        while True:
            counter += 1
            try:
                AXWorkflowEvent.save_workflow_event_to_db(workflow_id=ADC_WORKFLOW_ID,
                                                          event_type=AXWorkflowEvent.START,
                                                          detail={"version": __version__})
                return
            except Exception:
                logger.exception("[adc] save_workflow_start_event_to_db %s/%s", counter, max_retry)
                if counter > max_retry:
                    raise
                time.sleep(20)

    def workflow_d3_1_format(self, workflow_id):
        logger.info("[adc] [%s] workflow_d3_format v3", workflow_id)
        with self._workflow_sets_mutex:
            found = False
            for state in [AXWorkflow.RUNNING, AXWorkflow.RUNNING_DEL, AXWorkflow.RUNNING_DEL_FORCE]:
                if workflow_id in self._workflow_sets.get(state, {}):
                    found = True
                    break
        if found:
            ret = self.workflow_d3_3_format(workflow_id)
            if ret:
                return ret
        return self.workflow_d3_2_format(workflow_id)

    @staticmethod
    def workflow_d3_2_format(workflow_id):
        logger.info("[adc] [%s] workflow_d3_format v1", workflow_id)
        executor = AXWorkflowExecutor(workflow_id, "", "", fake_run=True)
        executor.init()
        executor._build_nodes()
        executor._recover()
        ret = executor._root_node.d3_format()
        ret["source_type"] = "replay: "
        return ret

    def workflow_d3_3_format(self, workflow_id):
        logger.info("[adc] [%s] workflow_d3_format v2", workflow_id)
        self._workflows_info[workflow_id] = {}
        workflow_query_key = AXWorkflow.REDIS_QUERY_LIST_KEY.format(workflow_id)
        redis_client.rpush(key=workflow_query_key,
                           value={},
                           expire=120,
                           encoder=json.dumps)
        count = 0
        max_count = 100
        try:
            while True:
                count += 1
                if (count > max_count) or len(self._workflows_info[workflow_id]) > 0:
                    ret = self._workflows_info.pop(workflow_id, {})
                    ret = ret.get("nodes", {})
                    ret["source_type"] = "live: "
                    return ret
                else:
                    time.sleep(0.5)
        except Exception:
            logger.exception("[adc] [%s] cannot format", workflow_id)
            return {}

    @staticmethod
    def workflow_event_show(node_id):
        if node_id is None:
            logger.error("[adc]: Reject workflow event show without id")
            raise AXIllegalArgumentException("[adc]: Workflow event show request does't have id.")

        logger.info("[adc] [%s]: show event request", node_id)
        node_events = axdb_client.get_node_events(root_id=node_id)
        if node_events:
            return {"events": node_events}
        else:
            return {"events": axdb_client.get_node_events(leaf_id=node_id)}
