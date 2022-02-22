import {ArchivedWorkflowsService} from './archived-workflows-service';
import {ClusterWorkflowTemplateService} from './cluster-workflow-template-service';
import {CronWorkflowService} from './cron-workflow-service';
import {EventService} from './event-service';
import {EventSourceService} from './event-source-service';
import {InfoService} from './info-service';
import {PipelineService} from './pipeline-service';
import {SensorService} from './sensor-service';
import {WorkflowTemplateService} from './workflow-template-service';
import {WorkflowsService} from './workflows-service';

interface Services {
    info: InfoService;
    sensor: SensorService;
    pipeline: PipelineService;
    event: EventService;
    eventSource: EventSourceService;
    workflows: WorkflowsService;
    workflowTemplate: WorkflowTemplateService;
    clusterWorkflowTemplate: ClusterWorkflowTemplateService;
    archivedWorkflows: ArchivedWorkflowsService;
    cronWorkflows: CronWorkflowService;
}

export const services: Services = {
    info: new InfoService(),
    workflows: new WorkflowsService(),
    workflowTemplate: new WorkflowTemplateService(),
    clusterWorkflowTemplate: new ClusterWorkflowTemplateService(),
    event: new EventService(),
    eventSource: new EventSourceService(),
    sensor: new SensorService(),
    pipeline: new PipelineService(),
    archivedWorkflows: new ArchivedWorkflowsService(),
    cronWorkflows: new CronWorkflowService()
};
