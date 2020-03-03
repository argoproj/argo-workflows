import {ArchivedWorkflowsService} from './archived-workflows-service';
import {CronWorkflowService} from './cron-workflow-service';
import {InfoService} from './info-service';
import {WorkflowTemplateService} from './workflow-template-service';
import {WorkflowsService} from './workflows-service';

export interface Services {
    info: InfoService;
    workflows: WorkflowsService;
    workflowTemplate: WorkflowTemplateService;
    archivedWorkflows: ArchivedWorkflowsService;
    cronWorkflows: CronWorkflowService;
}

export * from './workflows-service';
export * from './responses';

export const services: Services = {
    info: new InfoService(),
    workflows: new WorkflowsService(),
    workflowTemplate: new WorkflowTemplateService(),
    archivedWorkflows: new ArchivedWorkflowsService(),
    cronWorkflows: new CronWorkflowService()
};
