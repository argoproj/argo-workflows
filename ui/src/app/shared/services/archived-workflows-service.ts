import * as models from '../../../models';
import {Pagination} from '../pagination';
import {Utils} from '../utils';
import requests from './requests';
export class ArchivedWorkflowsService {
    public list(
        namespace: string,
        name: string,
        namePrefix: string,
        phases: string[],
        labels: string[],
        minStartedAt: Date,
        maxStartedAt: Date,
        pagination: Pagination,
        fields = [
            'metadata',
            'items.metadata.uid',
            'items.metadata.name',
            'items.metadata.namespace',
            'items.metadata.creationTimestamp',
            'items.metadata.labels',
            'items.metadata.annotations',
            'items.status.phase',
            'items.status.message',
            'items.status.finishedAt',
            'items.status.startedAt',
            'items.status.estimatedDuration',
            'items.status.progress',
            'items.spec.suspend'
        ]
    ) {
        const params = Utils.queryParams({
            name,
            namePrefix,
            phases,
            labels,
            minStartedAt,
            maxStartedAt,
            pagination
        });
        params.push(`fields=${fields.join(',')}`);
        if (namespace === '') {
            return requests.get(`api/v1/archived-workflows?${params.join('&')}`).then(res => res.body as models.WorkflowList);
        } else {
            return requests.get(`api/v1/archived-workflows?namespace=${namespace}&${params.join('&')}).join('&')}`).then(res => res.body as models.WorkflowList);
        }
    }

    public get(uid: string, namespace: string) {
        if (namespace === '') {
            return requests.get(`api/v1/archived-workflows/${uid}`).then(res => res.body as models.Workflow);
        } else {
            return requests.get(`api/v1/archived-workflows/${uid}?namespace=${namespace}`).then(res => res.body as models.Workflow);
        }
    }

    public delete(uid: string, namespace: string) {
        if (namespace === '') {
            return requests.delete(`api/v1/archived-workflows/${uid}`);
        } else {
            return requests.delete(`api/v1/archived-workflows/${uid}?namespace=${namespace}`);
        }
    }

    public listLabelKeys(namespace: string) {
        if (namespace === '') {
            return requests.get(`api/v1/archived-workflows-label-keys`).then(res => res.body as models.Labels);
        } else {
            return requests.get(`api/v1/archived-workflows-label-keys?namespace=${namespace}`).then(res => res.body as models.Labels);
        }
    }

    public async listLabelValues(key: string, namespace: string): Promise<models.Labels> {
        let url = `api/v1/archived-workflows-label-values?listOptions.labelSelector=${key}`;
        if (namespace !== '') {
            url += `&namespace=${namespace}`;
        }
        return (await requests.get(url)).body as models.Labels;
    }

    public resubmit(uid: string, namespace: string) {
        return requests
            .put(`api/v1/archived-workflows/${uid}/resubmit`)
            .send({namespace})
            .then(res => res.body as models.Workflow);
    }

    public retry(uid: string, namespace: string) {
        return requests
            .put(`api/v1/archived-workflows/${uid}/retry`)
            .send({namespace})
            .then(res => res.body as models.Workflow);
    }
}
