import * as models from '../../../models';
import requests from './requests';

export class ArchivedWorkflowsService {
    public list(namespace: string, continueArg: string) {
        return requests
            .get(`api/v1/archived-workflows?listOptions.fieldSelector=metadata.namespace=${namespace}&listOptions.continue=${continueArg}`)
            .then(res => res.body as models.WorkflowList);
    }

    public get(uid: string) {
        return requests.get(`api/v1/archived-workflows/${uid}`).then(res => res.body as models.Workflow);
    }

    public delete(uid: string) {
        return requests.delete(`api/v1/archived-workflows/${uid}`);
    }
}
