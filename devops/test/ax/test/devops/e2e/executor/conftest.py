import pytest


@pytest.fixture()
def workflow_executor():
    from ax.devops.workflow.ax_workflow_executor import AXWorkflowExecutor
    workflowexecutor = AXWorkflowExecutor("test_workflow_id", "test_name1", "URL")
    yield workflowexecutor
