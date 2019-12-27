import * as models from '../../../models';
import requests from './requests';

export class WorkflowHistoryService {
    public get(namespace: string, uid: string): Promise<models.Workflow> {
        return requests.get(`/workflow-history/${namespace}/${uid}`).then(res => res.body as models.Workflow);
    }

    public list(): Promise<models.Workflow[]> {
        return requests
            .get(`/workflow-history`)
            .then(res => res.body as models.WorkflowList)
            .then(list => list.items || []);
    }
}
