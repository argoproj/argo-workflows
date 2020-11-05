import {ArchivedWorkflowsService} from './archived-workflows-service';
import {ClusterWorkflowTemplateService} from './cluster-workflow-template-service';
import {CronWorkflowService} from './cron-workflow-service';
import {EventSourceService} from './event-source-service';
import {InfoService} from './info-service';
import {SensorService} from './sensor-service';
import {WorkflowTemplateService} from './workflow-template-service';
import {WorkflowsService} from './workflows-service';

export interface Services {
    info: InfoService;
    sensor: SensorService;
    eventSource: EventSourceService;
    workflows: WorkflowsService;
    workflowTemplate: WorkflowTemplateService;
    clusterWorkflowTemplate: ClusterWorkflowTemplateService;
    archivedWorkflows: ArchivedWorkflowsService;
    cronWorkflows: CronWorkflowService;
}

export * from './workflows-service';
export * from './responses';

export const services: Services = {
    info: new InfoService(),
    workflows: new WorkflowsService(),
    workflowTemplate: new WorkflowTemplateService(),
    clusterWorkflowTemplate: new ClusterWorkflowTemplateService(),
    eventSource: new EventSourceService(),
    sensor: new SensorService(),
    archivedWorkflows: new ArchivedWorkflowsService(),
    cronWorkflows: new CronWorkflowService()
};
