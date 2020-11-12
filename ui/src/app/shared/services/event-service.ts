import {WorkflowEventBindingList, WorkflowEventBindingWatchEvent} from '../../../models';
import requests from './requests';

export class EventService {
    public receiveEvent(namespace: string, discriminator: string, payload: any) {
        return requests.post(`api/v1/events/${namespace}/${discriminator}`).send(payload);
    }

    public listWorkflowEventBindings(namespace: string) {
        return requests.get(`api/v1/workflow-event-bindings/${namespace}`).then(res => res.body as WorkflowEventBindingList);
    }

    public watchWorkflowEventBindings(namespace: string, resourceVersion: string) {
        return requests
            .loadEventSource(`api/v1/stream/workflow-event-bindings/${namespace}?listOptions.resourceVersion=${resourceVersion}`)
            .map(line => JSON.parse(line).result as WorkflowEventBindingWatchEvent);
    }
}
