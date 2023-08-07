import {ClusterWorkflowTemplateService} from './cluster-workflow-template-service';
import {CronWorkflowService} from './cron-workflow-service';
import {EventService} from './event-service';
import {EventSourceService} from './event-source-service';
import {InfoService} from './info-service';
import {SensorService} from './sensor-service';
import {WorkflowTemplateService} from './workflow-template-service';
import {WorkflowsService} from './workflows-service';

interface Services {
    info: typeof InfoService;
    sensor: typeof SensorService;
    event: typeof EventService;
    eventSource: typeof EventSourceService;
    workflows: typeof WorkflowsService;
    workflowTemplate: typeof WorkflowTemplateService;
    clusterWorkflowTemplate: typeof ClusterWorkflowTemplateService;
    cronWorkflows: typeof CronWorkflowService;
}

export const services: Services = {
    info: InfoService,
    workflows: WorkflowsService,
    workflowTemplate: WorkflowTemplateService,
    clusterWorkflowTemplate: ClusterWorkflowTemplateService,
    event: EventService,
    eventSource: EventSourceService,
    sensor: SensorService,
    cronWorkflows: CronWorkflowService
};
