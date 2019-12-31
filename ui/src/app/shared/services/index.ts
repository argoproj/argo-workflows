import {WorkflowHistoryService} from './workflow-history-service';
import {WorkflowTemplateService} from './workflow-template-service';
import {WorkflowsService} from './workflows-service';

export interface Services {
    workflows: WorkflowsService;
    workflowTemplate: WorkflowTemplateService;
    workflowHistory: WorkflowHistoryService;
}

export * from './workflows-service';
export * from './responses';

export const services: Services = {
    workflows: new WorkflowsService(),
    workflowTemplate: new WorkflowTemplateService(),
    workflowHistory: new WorkflowHistoryService()
};
