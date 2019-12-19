import {WorkflowsService} from './workflows-service';

export interface Services {
    workflows: WorkflowsService;
}

export * from './workflows-service';
export * from './responses';

export const services: Services = {
    workflows: new WorkflowsService()
};
