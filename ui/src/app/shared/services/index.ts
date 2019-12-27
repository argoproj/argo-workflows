import {WorkflowHistoryService} from './workflow-history-service';
import {WorkflowsService} from './workflows-service';

export interface Services {
    workflows: WorkflowsService;
    workflowHistory: WorkflowHistoryService;
}

export * from './workflows-service';

export const services: Services = {
    workflows: new WorkflowsService(),
    workflowHistory: new WorkflowHistoryService()
};
