
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
from argo_workflows.api.archived_workflow_service_api import ArchivedWorkflowServiceApi
from argo_workflows.api.artifact_service_api import ArtifactServiceApi
from argo_workflows.api.cluster_workflow_template_service_api import ClusterWorkflowTemplateServiceApi
from argo_workflows.api.cron_workflow_service_api import CronWorkflowServiceApi
from argo_workflows.api.event_service_api import EventServiceApi
from argo_workflows.api.event_source_service_api import EventSourceServiceApi
from argo_workflows.api.info_service_api import InfoServiceApi
from argo_workflows.api.sensor_service_api import SensorServiceApi
from argo_workflows.api.workflow_service_api import WorkflowServiceApi
from argo_workflows.api.workflow_template_service_api import WorkflowTemplateServiceApi
