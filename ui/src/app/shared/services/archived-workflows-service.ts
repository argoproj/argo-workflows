import * as models from '../../../models';
import {Pagination} from '../pagination';
import {Utils} from '../utils';
import requests from './requests';
export class ArchivedWorkflowsService {
    public list(namespace: string, name: string, namePrefix: string, phases: string[], labels: string[], minStartedAt: Date, maxStartedAt: Date, pagination: Pagination) {
        return requests
            .get(`api/v1/archived-workflows?${Utils.queryParams({namespace, name, namePrefix, phases, labels, minStartedAt, maxStartedAt, pagination}).join('&')}`)
            .then(res => res.body as models.WorkflowList);
    }

    public get(uid: string) {
        return requests.get(`api/v1/archived-workflows/${uid}`).then(res => res.body as models.Workflow);
    }

    public delete(uid: string) {
        return requests.delete(`api/v1/archived-workflows/${uid}`);
    }

    public listLabelKeys() {
        return requests.get(`api/v1/archived-workflows-label-keys`).then(res => res.body as models.Labels);
    }

    public listLabelValues(key: string) {
        return requests.get(`api/v1/archived-workflows-label-values?listOptions.labelSelector=${key}`).then(res => res.body as models.Labels);
    }
}
