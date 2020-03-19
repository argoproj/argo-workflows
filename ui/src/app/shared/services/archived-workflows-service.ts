import * as models from '../../../models';
import requests from './requests';

export class ArchivedWorkflowsService {
    public list(namespace: string, phases: string[], labels: string[], minStartedAt: Date, maxStartedAt: Date, offset: number) {
        return requests
            .get(`api/v1/archived-workflows?${this.queryParams({namespace, phases, labels, minStartedAt, maxStartedAt, offset}).join('&')}`)
            .then(res => res.body as models.WorkflowList);
    }

    public get(uid: string) {
        return requests.get(`api/v1/archived-workflows/${uid}`).then(res => res.body as models.Workflow);
    }

    public delete(uid: string) {
        return requests.delete(`api/v1/archived-workflows/${uid}`);
    }

    private queryParams(filter: {namespace?: string; phases?: Array<string>; labels?: Array<string>; minStartedAt?: Date; maxStartedAt?: Date; offset?: number}) {
        const queryParams: string[] = [];
        const fieldSelector = this.fieldSelectorParams(filter.namespace, filter.minStartedAt, filter.maxStartedAt);
        if (fieldSelector.length > 0) {
            queryParams.push(`listOptions.fieldSelector=${fieldSelector}`);
        }
        const labelSelector = this.labelSelectorParams(filter.phases, filter.labels);
        if (labelSelector.length > 0) {
            queryParams.push(`listOptions.labelSelector=${labelSelector}`);
        }
        if (filter.offset) {
            queryParams.push(`listOptions.continue=${filter.offset}`);
        }
        return queryParams;
    }

    private fieldSelectorParams(namespace: string, minStartedAt: Date, maxStartedAt: Date) {
        let fieldSelector = '';
        if (namespace) {
            fieldSelector += 'metadata.namespace=' + namespace + ',';
        }
        if (minStartedAt) {
            fieldSelector += 'spec.startedAt>' + minStartedAt.toISOString() + ',';
        }
        if (maxStartedAt) {
            fieldSelector += 'spec.startedAt<' + maxStartedAt.toISOString() + ',';
        }
        if (fieldSelector.endsWith(',')) {
            fieldSelector = fieldSelector.substr(0, fieldSelector.length - 1);
        }
        return fieldSelector;
    }

    private labelSelectorParams(phases?: Array<string>, labels?: Array<string>) {
        let labelSelector = '';
        if (phases && phases.length > 0) {
            labelSelector = `workflows.argoproj.io/phase in (${phases.join(',')})`;
        }
        if (labels && labels.length > 0) {
            if (labelSelector.length > 0) {
                labelSelector += ',';
            }
            labelSelector += labels.join(',');
        }
        return labelSelector;
    }
}
