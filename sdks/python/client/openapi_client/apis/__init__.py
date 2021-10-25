
# flake8: noqa

# Import all APIs into this package.
# If you have many APIs here with many many models used in each API this may
# raise a `RecursionError`.
# In order to avoid this, import only the API that you directly need like:
#
#   from .api.archived_workflow_service_api import ArchivedWorkflowServiceApi
#
# or import this package, but before doing it, use:
#
#   import sys
#   sys.setrecursionlimit(n)

# Import APIs into API package:
from io.argoproj.workflow.apis.archived_workflow_service_api import ArchivedWorkflowServiceApi
from io.argoproj.workflow.apis.artifact_service_api import ArtifactServiceApi
from io.argoproj.workflow.apis.cluster_workflow_template_service_api import ClusterWorkflowTemplateServiceApi
from io.argoproj.workflow.apis.cron_workflow_service_api import CronWorkflowServiceApi
from io.argoproj.workflow.apis.event_service_api import EventServiceApi
from io.argoproj.workflow.apis.event_source_service_api import EventSourceServiceApi
from io.argoproj.workflow.apis.info_service_api import InfoServiceApi
from io.argoproj.workflow.apis.pipeline_service_api import PipelineServiceApi
from io.argoproj.workflow.apis.sensor_service_api import SensorServiceApi
from io.argoproj.workflow.apis.workflow_service_api import WorkflowServiceApi
from io.argoproj.workflow.apis.workflow_template_service_api import WorkflowTemplateServiceApi
