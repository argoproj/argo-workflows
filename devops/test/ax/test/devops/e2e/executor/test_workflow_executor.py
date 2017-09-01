import logging
import os
import json
import pytest

from ax.devops.workflow.ax_workflow_executor import AXWorkflowNodeResult, AXWorkflowEvent, AXWorkflowExecutor

logger = logging.getLogger(__name__)
logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(name)s %(lineno)d %(threadName)s: %(message)s")


def get_json(name):
    script_dir = os.path.dirname(__file__)
    rel_path = 'resources/{}.json'.format(name)
    abs_file_path = os.path.join(script_dir, rel_path)
    with open(abs_file_path, 'r') as the_file:
        a = json.load(the_file)
        return a


def save_to_db(leaf_result):
    leaf_service = {
        'leaf_id': leaf_result.node_id,
        'root_id': leaf_result.workflow_id,
        'sn': leaf_result.sn,
        'result': leaf_result.result_code,
        'timestamp': leaf_result.timestamp,
        'detail': leaf_result.detail
    }
    with open("test2_recover_results.txt", "a+") as file:
        file.write(json.dumps(leaf_service))
        file.write('\n')


def return_success(*args, node, root_service_template):
    # time.sleep(random.randint(0, 1))
    return "SUCCEED", None


@pytest.mark.parametrize("name", ['test1', 'test2'])
def test_build(workflow_executor, monkeypatch, name):
    logger.info('\n\nTest workflow node Build using test resource, %s', name)

    monkeypatch.setattr(workflow_executor, '_service_template', get_json(name))
    workflow_executor._build_nodes()
    root = workflow_executor._root_node
    build_res = get_json("{}_build".format(name))

    # Verify the state of workflow tree after build
    assert root.jsonify() == build_res


@pytest.mark.parametrize("name", ['test1', 'test2'])
def test_start(workflow_executor, monkeypatch, name):
    logger.info('\n\nTest workflow node Start using test resource, %s', name)

    monkeypatch.setattr(workflow_executor, '_service_template', get_json(name))
    monkeypatch.setattr(workflow_executor, 'last_step', lambda *args, **kwargs: None)
    monkeypatch.setattr(workflow_executor, 'start_start_and_monitor_container_thread', lambda *args, **kwargs: None)
    monkeypatch.setattr(workflow_executor, '_wait_for_state_to_be_running_or_running_del_or_done', lambda *args, **kwargs: None)
    monkeypatch.setattr(workflow_executor, 'do_report_to_kafka', lambda *args, **kwargs: None)

    workflow_executor._build_nodes()
    root = workflow_executor._root_node

    start_res = get_json("{}_start".format(name))

    # Verify the state of workflow tree after start
    workflow_executor._start_if_have_not()

    print(json.dumps(root.jsonify(), indent=2))
    assert root.jsonify() == start_res


@pytest.mark.parametrize("name", ["test1", "test2"])
def test_recover(workflow_executor, monkeypatch, name):
    def load_from_db(*args):
        script_dir = os.path.dirname(__file__)
        node_filename = "resources/{}_recover_results.txt".format(name)
        abs_file_path = os.path.join(script_dir, node_filename)
        if os.path.exists(abs_file_path):
            with open(abs_file_path) as data_in:
                lines = [line.rstrip('\n') for line in data_in]
                results = []
                for line in lines:
                    r = json.loads(line)
                    from ax.devops.workflow.ax_workflow_executor import AXWorkflowNodeResult
                    results.append(AXWorkflowNodeResult(workflow_id=r.get("root_id"),
                                                        node_id=r.get("leaf_id"),
                                                        sn=r.get("sn"),
                                                        detail=r.get("detail", None),
                                                        result_code=r.get("result"),
                                                        timestamp=r.get("timestamp", None)))
        else:
            return []
        return sorted(results, key=lambda result: result.sn)

    logger.info('\n\nTest workflow node Recovery using test resource, %s', name)

    monkeypatch.setattr(workflow_executor, '_start_and_monitor_container', return_success)
    monkeypatch.setattr(workflow_executor, '_service_template', get_json(name))
    monkeypatch.setattr(workflow_executor, '_get_workflow_results_from_db', load_from_db)

    monkeypatch.setattr(workflow_executor, '_wait_for_state_to_be_running_or_running_del_or_done', lambda *args, **kwargs: None)
    monkeypatch.setattr(workflow_executor, 'last_step_update_db', lambda *args, **kwargs: None)
    monkeypatch.setattr(workflow_executor, 'stop_self_container', lambda *args, **kwargs: None)
    monkeypatch.setattr(workflow_executor, 'report_to_adc', lambda *args, **kwargs: None)
    monkeypatch.setattr(workflow_executor, 'do_report_to_kafka', lambda *args, **kwargs: None)
    monkeypatch.setattr(workflow_executor, '_get_workflow_node_events_from_db', lambda *args, **kwargs: None)

    monkeypatch.setattr(AXWorkflowNodeResult, 'save_result_to_db_helper', lambda *args, **kwargs: None)
    monkeypatch.setattr(AXWorkflowEvent, '_save_workflow_event_to_db_helper', lambda *args, **kwargs: None)

    workflow_executor._build_nodes()
    root = workflow_executor._root_node
    workflow_executor._recover()

    # Verify the state of the workflow tree after recover
    recover_res = get_json("{}_recover".format(name))
    assert root.jsonify() == recover_res

    workflow_executor._start_if_have_not()
    workflow_executor._wait_and_process_results()

    # Verify the state of the root after workflow complete
    assert root.is_succeed


@pytest.mark.parametrize("name, result_expected, leaf_expected",
                         [('test1', [0, 0], [0, 0]),
                          ('test2', [0.088, 44.0], [0.088, 44.0]),
                          ('test3', [0.0798, 94.38], [0.026, 29.53]),
                          ('test4', [0.380, 91.34], [0.380, 91.34])])
def test_resource(workflow_executor, monkeypatch, name, result_expected, leaf_expected):
    logger.info('\n\nTest workflow node Start using test resource, %s', name)

    monkeypatch.setattr(workflow_executor, '_service_template', get_json(name))
    workflow_executor._build_nodes()
    root = workflow_executor._root_node

    resource_list = root.max_resource
    leaf_resource_list = root.max_leaf_resource
    print(resource_list.resource)
    print(leaf_resource_list.resource)
    for i in range(2):
        assert abs(resource_list.resource[i] - result_expected[i]) < 0.001
        assert abs(leaf_resource_list.resource[i] - leaf_expected[i]) < 0.001


def test_resource_scaler1(workflow_executor):
    cpu, mem = AXWorkflowExecutor.process_leaf_node_resource(cpu_core=1,
                                                             mem_mib=100,
                                                             disk_gb=0,
                                                             instance_cpu_core=2,
                                                             instance_mem_mib=500,
                                                             instance_disk_gb=100,
                                                             ax_cpu_core=0.3,
                                                             ax_mem_mib=100,
                                                             sidecar_cpu_core=0.1,
                                                             sidecar_mem_mib=100)
    print(cpu)
    print(mem)
    assert cpu == 0.75
    assert mem == 100
