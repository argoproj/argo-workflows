
from ax.util.az_patch import az_patch
az_patch()

import json
import logging
import math
import os
import random
import time
import unittest
import uuid

from unittest.mock import patch

import ax.devops.apps.workflow.test_ax_workflow as test_ax_workflow

from .ax_workflow import AXWorkflow
from .ax_workflow_executor import AXWorkflowExecutor, AXWorkflowNodeResult

logger = logging.getLogger(__name__)


def _get_testmode_node_state_file(workflow_id):
    return workflow_id + ".node"


def gen_random_workflow(parameter,
                        service_id, name_prefix, depth,
                        is_fixture_leaf=False,
                        is_static_fixture=False,
                        has_artifact_output=False,
                        artifact_input_id=None):
    max_depth = (parameter and parameter.get("max_depth", 0)) or 2
    max_length = (parameter and parameter.get("max_length", 0)) or 2
    max_width = (parameter and parameter.get("max_width", 0)) or 2
    max_cpu = (parameter and parameter.get("max_cpu", 0)) or 0.02
    max_disk = (parameter and parameter.get("max_disk", 0)) or 0.0
    max_mem = (parameter and parameter.get("max_mem", 0)) or 64.0
    step_duration_second = (parameter and parameter.get("step_duration_second", 10)) or 10
    failure_rate = (parameter and parameter.get("failure_rate", 0)) or 0
    dind_rate = (parameter and parameter.get("dind_rate", 0)) or 500
    fixture_failure_rate = (parameter and parameter.get("fixture_failure_rate", 0)) or 0
    image = (parameter and parameter.get("image")) or "alpine:3.5"
    always_run_rate = (parameter and parameter.get("always_run_rate", 0)) or 0
    ignore_error_rate = (parameter and parameter.get("ignore_error_rate", 0)) or 0
    skipped_rate = (parameter and parameter.get("skipped_rate", 0)) or 0
    deployment_rate = (parameter and parameter.get("deployment_rate", 0)) or 0
    deployment_life = (parameter and parameter.get("deployment_life", 20)) or 20

    failure_node_count = 0
    total_node_count = 1
    leaf_node_count = 0
    r = {
        "id": service_id,
        "costid": {
            "app": "workflow",
            "project": "prod"
        },
        "template": {"id": str(uuid.uuid4()),
                     "type": "service_template",
                     "subtype": "workflow",
                     "version": "1.0",
                     "name": name_prefix,
                     "dns_name": "",
                     "description": "",
                     "cost": 0
                     },
        "flags": {}
    }

    if (random.randint(0, 1000) < always_run_rate):
        r["flags"]["always_run"] = True

    if (random.randint(0, 1000) < ignore_error_rate):
        r["flags"]["ignore_error"] = True

    if (random.randint(0, 1000) < skipped_rate):
        r["flags"]["skipped"] = True

    if (random.randint(0, max_depth) >= depth or depth <= 2) and (depth < max_depth) and not is_fixture_leaf and not has_artifact_output:
        # add steps and fixtures
        for f_or_s in ["steps", "fixtures"]:
            is_fixtures = (f_or_s == "fixtures")
            r["template"][f_or_s] = []
            step_num = random.randint(0, max_length)
            if depth < 2:
                step_num = max_length
            for i in range(0, step_num):
                step = {}
                parallel_step = random.randint(max_width - depth, max_width)
                gen_artifact_output = False
                if f_or_s == "steps" and depth == 1 and i == 0:
                    parallel_step = 1
                    gen_artifact_output = True
                if parallel_step < 0:
                    parallel_step = -parallel_step
                if is_fixtures and random.randint(0, 5) < 3:
                    is_static_fixture = True
                else:
                    is_static_fixture = False
                for j in range(0, parallel_step):
                    name = "{}-{}-{}".format(name_prefix, i, j)
                    if is_fixtures:
                        name = "f-" + name
                        if is_static_fixture:
                            name = "f" + name
                    service_id = str(uuid.uuid4())
                    if gen_artifact_output:
                        artifact_input_id = service_id
                    step[name], node_count, failure_node, leaf_node = gen_random_workflow(parameter=parameter,
                                                                                          service_id=service_id,
                                                                                          name_prefix=name,
                                                                                          depth=depth + 1,
                                                                                          is_fixture_leaf=is_fixtures,
                                                                                          is_static_fixture=is_static_fixture,
                                                                                          has_artifact_output=gen_artifact_output,
                                                                                          artifact_input_id=artifact_input_id if f_or_s == "steps" else None)
                    total_node_count += node_count
                    failure_node_count += failure_node
                    leaf_node_count += leaf_node
                r["template"][f_or_s].append(step)
    else:
        # r is leaf
        r["template"]["container"] = {
            "instanceid": str(uuid.uuid4()),
            "id": name_prefix,
            "instances": 1,
            "tags": {
                "test": [
                    "notuse"
                ]
            },
            "constraints": [
                ["hostname", "UNLIKE", "dns"]
            ],
            "resources": {
                "cpu_cores": float("{0:.2f}".format(random.uniform(max_cpu/2, max_cpu))),
                "disk_gb": float("{0:.2f}".format(random.uniform(max_disk/2, max_disk))),
                "mem_mib": math.ceil(max(4.0, random.uniform(max_mem/2, max_mem)))
            },
            "image": image,
            "entrypoint": "/bin/sh",
            "command": ["-c"]
        }

        artifact_check_cmd = ""
        if has_artifact_output:
            r["template"]["outputs"] = {
                "artifacts": {"axsrc": { "path": "/ax/src" },
                              "axsingle": {"path": "/etc/hosts"},
                              }
            }
        elif artifact_input_id:
            r["template"]["inputs"] = {
                          "artifacts": [
                              {
                                  "name": "axsrc",
                                  "path": "/inputs",
                                  "service_instance_id": artifact_input_id
                              },
                              {
                                  "name": "axsingle",
                                  "path": "/inputs_single/single",
                                  "service_instance_id": artifact_input_id
                              },
                          ]
                      }
            artifact_check_cmd = "ls -lrt /inputs && ls -lrt /inputs_single/single &&"
        is_deployment = False
        is_dind = False
        if is_fixture_leaf:
            real_failure_rate = 0 if is_static_fixture else fixture_failure_rate
            # no flags for fixtures
            r["flags"] = {}
        else:
            real_failure_rate = failure_rate
            if random.randint(0, 1000) < deployment_rate:
                is_deployment = True
                label = { "ax_ea_deployment": "{ \"ports\": [ { \"name\": \"abc\", \"port\": 12345, \"containerPort\": 12345 } ] }"}
                r["template"].setdefault("labels", {}).update(label)
            if random.randint(0, 1000) < dind_rate:
                is_dind = True
                r["template"]["container"]["resources"]["mem_mib"] *= 2
                label = {"ax_ea_docker_enable": "{ \"graph-storage-name\": \"randteststorage\", \"graph-storage-size\": \"1Gi\" }"}
                if random.randint(0, 1000) < dind_rate:
                    label = {"ax_ea_docker_enable": "{ \"graph-storage-name\": \"randteststorage\", \"graph-storage-size\": \"1Gi\", \"cpu_cores\": \"0.01\", \"mem_mib\": \"32\"}"}
                r["template"].setdefault("labels", {}).update(label)

        if random.randint(0, 1000) < real_failure_rate:
            result_exec = "false"
            failure_node_count += 1
        else:
            result_exec = "true"

        if (is_fixture_leaf and result_exec == "true"):
            effective_container_duration = step_duration_second * (10 ** (random.randint(0, 40) / 10.0))
        elif is_deployment:
            effective_container_duration = step_duration_second * (10 ** (random.randint(0, deployment_life) / 10.0))
        else:
            effective_container_duration = random.randint(step_duration_second // 2, step_duration_second)

        run_cmd = "sleep {}".format(effective_container_duration)
        # run_cmd = "a=0; while true; do let a+=1; echo $a 12345678901234567890; done"
        if is_dind:
            run_cmd = "docker version && {} ".format(run_cmd)

        cmd = "{} {} && {}".format(artifact_check_cmd, run_cmd, result_exec)

        r["template"]["container"]["command"].append(cmd)
        leaf_node_count += 1
        if is_fixture_leaf and is_static_fixture:
            r = {
                "id": service_id,
                "requirements":
                    {
                        "is_ax_test_mock": True,
                        "name": "ax_mock_name",
                        "class": "ax_mock",
                        "attributes": {
                            "mock_attr": 123
                        }
                     }
            }
    return r, total_node_count, failure_node_count, leaf_node_count


def _get_testmode_random_service_tempate(workflow_id, image):
    parameter = {
        "max_depth": 2,
        "max_width": 2,
        "max_length": 2,
        "max_cpu": 0.03,
        "max_disk": 0.0,
        "max_mem": 64.0,
        "step_duration_second": 20,
        "failure_rate": 0,
        "dind_rate": 500,
        "image": image
    }
    a, count, failure_count, leaf_count = gen_random_workflow(parameter=parameter,
                                                              service_id=workflow_id,
                                                              name_prefix="root",
                                                              depth=1)
    print("total_count={} failure_count={} leaf_count={}".format(count, failure_count, leaf_count))

    return a


def test_load_results_from_db(workflow_id):
    node_filename = _get_testmode_node_state_file(workflow_id)
    if os.path.exists(node_filename):
        with open(node_filename) as data_in:
            lines = [line.rstrip('\n') for line in data_in]
            results = []
            for line in lines:
                r = json.loads(line)
                results.append(AXWorkflowNodeResult(workflow_id=r.get("workflow_id"),
                                                    node_id=r.get("node_id"),
                                                    sn=r.get("sn"),
                                                    detail=r.get("detail", None),
                                                    result=r.get("result"),
                                                    timestamp=r.get("timestamp", None)))
    else:
        return []
    return sorted(results, key=lambda result: result.sn)


def test_start_and_monitor_container(node):
    # start and monitor container
    time.sleep(random.randint(0, 1000) / 1000)
    if False and random.randint(0, 1000) == 1:
        result = AXWorkflowNodeResult.INTERRUPTED
    elif False and random.randint(0, 1000) == 1:
        result = AXWorkflowNodeResult.FAILED
    else:
        result = AXWorkflowNodeResult.SUCCEED
    return result


def test_save_result_to_db_helper(leaf_id, workflow_id, sn, result, timestamp, detail):
    data = {"workflow_id": workflow_id,
            "node_id": leaf_id,
            "sn": sn,
            "result": result,
            "timestamp": timestamp,
            "detail": detail}
    with open(_get_testmode_node_state_file(workflow_id), "a+") as myfile:
        myfile.write(json.dumps(data))
        myfile.write('\n')


def test_get_workflow_by_id_from_db(workflow_id, need_load_template=False):
    workflow = test_ax_workflow.get_workflow_by_id_from_db(workflow_id, need_load_template=need_load_template)

    if workflow is None:
        image = "alpine:3.5"
        data = _get_testmode_random_service_tempate(workflow_id, image)
        workflow = AXWorkflow(workflow_id=workflow_id, service_template=data,
                              status=AXWorkflow.RUNNING, timestamp=AXWorkflow.get_current_epoch_timestamp_in_ms())
        assert AXWorkflow.post_workflow_to_db(workflow=workflow)
        node_filename = _get_testmode_node_state_file(workflow_id)
        if os.path.exists(node_filename):
            os.remove(node_filename)

    return workflow


class TestAXWorkflowExecutor(unittest.TestCase):
    @patch('ax_workflow_executor.AXWorkflowNodeResult.load_results_from_db', side_effect=test_load_results_from_db)
    @patch('ax_workflow_executor.AXWorkflowExecutor._start_and_monitor_container_helper',
           side_effect=test_start_and_monitor_container)
    @patch('ax_workflow_executor.AXWorkflowNodeResult.save_result_to_db_helper',
           side_effect=test_save_result_to_db_helper)
    @patch('ax_workflow.AXWorkflow.get_workflow_by_id_from_db', side_effect=test_get_workflow_by_id_from_db)
    @patch('ax_workflow.AXWorkflow.post_workflow_to_db', side_effect=test_ax_workflow.post_workflow_to_db)
    @patch('ax_workflow.AXWorkflow.update_workflow_status_in_db',
           side_effect=test_ax_workflow.update_workflow_status_in_db)
    @patch('ax_workflow.AXWorkflow.get_workflows_by_status_from_db',
           side_effect=test_ax_workflow.get_workflows_by_status_from_db)
    def testRunExecutor(self, func1, func2, func3, func4, func5, func6, func7):
        exe = AXWorkflowExecutor(workflow_id="5e948d93-1b92-11e6-8140-0242ac110002")
        exe.run()


# class TestADC(unittest.TestCase):
#     @patch('ax_workflow_executor.AXWorkflowNodeResult.load_results_from_db', side_effect=test_load_results_from_db)
#     @patch('ax_workflow_executor.AXWorkflowExecutor._start_and_monitor_container_helper', side_effect=test_start_and_monitor_container)
#     @patch('ax_workflow_executor.AXWorkflowNodeResult.save_result_to_db_helper', side_effect=test_save_result_to_db_helper)
#     @patch('ax_workflow.AXWorkflow.get_workflow_by_id_from_db', side_effect=test_get_workflow_by_id_from_db)
#     @patch('ax_workflow.AXWorkflow.post_workflow_to_db', side_effect= test_ax_workflow.post_workflow_to_db)
#     @patch('ax_workflow.AXWorkflow.update_workflow_status_in_db', side_effect=test_ax_workflow.update_workflow_status_in_db)
#     @patch('ax_workflow.AXWorkflow.get_workflows_by_status_from_db', side_effect=test_ax_workflow.get_workflows_by_status_from_db)
#     def testADC(self, func1, func2, func3, func4, func5, func6, func7):
#         from adc_main import ADC, __version__
#         from adc_rest import adc_rest_start
#
#         logger = logging.getLogger('axdevops.adc.adc_entry')
#
#         # Basic logging.
#         logging.basicConfig(format="%(asctime)s %(levelname)s %(name)s %(lineno)d %(threadName)s: %(message)s")
#         logging.getLogger("axdevops").setLevel(logging.DEBUG)
#         logging.getLogger("transitions").setLevel(logging.INFO)
#
#         adc_rest_start(port=None)
#         ADC().run()
#
# testAXWorkflowExecutor = TestAXWorkflowExecutor
# testADC = TestADC

if __name__ == '__main__':
    unittest.main()
