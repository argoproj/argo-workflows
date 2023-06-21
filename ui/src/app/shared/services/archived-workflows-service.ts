import * as models from '../../../models';
import {Pagination} from '../pagination';
import {Utils} from '../utils';
import requests from './requests';
export const ArchivedWorkflowsService = {
    list(namespace: string, name: string, namePrefix: string, phases: string[], labels: string[], minStartedAt: Date, maxStartedAt: Date, pagination: Pagination) {
        if (namespace === '') {
            return requests
                .get(`api/v1/archived-workflows?${Utils.queryParams({name, namePrefix, phases, labels, minStartedAt, maxStartedAt, pagination}).join('&')}`)
                .then(res => res.body as models.WorkflowList);
        } else {
            return requests
                .get(`api/v1/archived-workflows?namespace=${namespace}&${Utils.queryParams({name, namePrefix, phases, labels, minStartedAt, maxStartedAt, pagination}).join('&')}`)
                .then(res => res.body as models.WorkflowList);
        }
    },

    get(uid: string, namespace: string) {
        if (namespace === '') {
            return requests.get(`api/v1/archived-workflows/${uid}`).then(res => res.body as models.Workflow);
        } else {
            return requests.get(`api/v1/archived-workflows/${uid}?namespace=${namespace}`).then(res => res.body as models.Workflow);
        }
    },

    delete(uid: string, namespace: string) {
        if (namespace === '') {
            return requests.delete(`api/v1/archived-workflows/${uid}`);
        } else {
            return requests.delete(`api/v1/archived-workflows/${uid}?namespace=${namespace}`);
        }
    },

    listLabelKeys(namespace: string) {
        if (namespace === '') {
            return requests.get(`api/v1/archived-workflows-label-keys`).then(res => res.body as models.Labels);
        } else {
            return requests.get(`api/v1/archived-workflows-label-keys?namespace=${namespace}`).then(res => res.body as models.Labels);
        }
    },

    async listLabelValues(key: string, namespace: string): Promise<models.Labels> {
        let url = `api/v1/archived-workflows-label-values?listOptions.labelSelector=${key}`;
        if (namespace !== '') {
            url += `&namespace=${namespace}`;
        }
        return (await requests.get(url)).body as models.Labels;
    },

    resubmit(uid: string, namespace: string) {
        return requests
            .put(`api/v1/archived-workflows/${uid}/resubmit`)
            .send({namespace})
            .then(res => res.body as models.Workflow);
    },

    retry(uid: string, namespace: string) {
        return requests
            .put(`api/v1/archived-workflows/${uid}/retry`)
            .send({namespace})
            .then(res => res.body as models.Workflow);
    }
};
