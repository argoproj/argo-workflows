import {WorkflowEventBindingList} from '../../../models';
import requests from './requests';

export class EventService {
    public listWorkflowEventBindings(namespace: string) {
        return requests.get(`api/v1/workflow-event-bindings/${namespace}`).then(res => res.body as WorkflowEventBindingList);
    }
}
