import * as models from '../../../models';
import requests from './requests';

export class WorkflowHistoryService {
    public list() {
        return requests
            .get(`/workflow-history`)
            .then(res => res.body as models.WorkflowList)
            .then(list => list.items || []);
    }

    public get(namespace: string, uid: string): Promise<models.Workflow> {
        return requests.get(`/workflow-history/${namespace}/${uid}`).then(res => res.body as models.Workflow);
    }

    public resubmit(namespace: string, uid: string) {
        return requests.put(`/workflow-history/${namespace}/${uid}/resubmit`).then(res => res.body as models.Workflow);
    }

    public delete(namespace: string, uid: string) {
        return requests.delete(`/workflow-history/${namespace}/${uid}`);
    }
}
