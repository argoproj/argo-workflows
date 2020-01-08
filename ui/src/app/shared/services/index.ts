import {ArchivedWorkflowsService} from './archived-workflows-service';
import {WorkflowTemplateService} from './workflow-template-service';
import {WorkflowsService} from './workflows-service';
import {CronWorkflowService} from "./cron-workflow-service";

export interface Services {
    workflows: WorkflowsService;
    workflowTemplate: WorkflowTemplateService;
    archivedWorkflows: ArchivedWorkflowsService;
    cronWorkflows: CronWorkflowService;
}

export * from './workflows-service';
export * from './responses';

export const services: Services = {
    workflows: new WorkflowsService(),
    workflowTemplate: new WorkflowTemplateService(),
    archivedWorkflows: new ArchivedWorkflowsService(),
    cronWorkflows: new CronWorkflowService()
};
