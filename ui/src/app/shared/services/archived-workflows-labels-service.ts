import * as models from '../../../models';
import requests from './requests';

export class ArchivedWorkflowsLabelsService {
    public list() {
        return requests.get(`api/v1/archived-workflows-labels/keys`).then(res => res.body as models.Labels);
    }

    public get(key: string) {
        return requests.get(`api/v1/archived-workflows-labels?${this.queryParams({key}).join('&')}`).then(res => res.body as models.Labels);
    }

    private queryParams(filter: {key?: string}) {
        const queryParams: string[] = [];
        const fieldSelector = this.fieldSelectorParams(filter.key);
        if (fieldSelector.length > 0) {
            queryParams.push(`listOptions.fieldSelector=${fieldSelector}`);
        }
        return queryParams;
    }

    private fieldSelectorParams(key: string) {
        let fieldSelector = '';
        if (key) {
            fieldSelector += 'key=' + key + ',';
        }
        return fieldSelector;
    }
}
