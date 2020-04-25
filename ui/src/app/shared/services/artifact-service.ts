import {WorkflowList} from '../../../models';
import requests from './requests';

export class ArtifactsService {
    public list(namespace: string, phases: string[], labels: string[]) {
        return requests.get(`api/v1/artifacts/${namespace}?${this.queryParams({phases, labels}).join('&')}`).then(res => res.body as WorkflowList);
    }

    private queryParams(filter: {namespace?: string; name?: string; phases?: Array<string>; labels?: Array<string>}) {
        const queryParams: string[] = [];
        if (filter.name) {
            queryParams.push(`listOptions.fieldSelector=metadata.name=${filter.name}`);
        }
        const labelSelector = this.labelSelectorParams(filter.phases, filter.labels);
        if (labelSelector.length > 0) {
            queryParams.push(`listOptions.labelSelector=${labelSelector}`);
        }
        return queryParams;
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
