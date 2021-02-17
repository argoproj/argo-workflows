import {WorkflowPhase} from '../../../models';

export function queryParams(filter: {namespace?: string; name?: string; phases?: Array<WorkflowPhase>; labels?: Array<string>; resourceVersion?: string}): any {
    const queryParamList: string[] = [];
    if (filter.name) {
        queryParamList.push(`listOptions.fieldSelector=metadata.name=${filter.name}`);
    }
    const labelSelector = this.labelSelectorParams(filter.phases, filter.labels);
    if (labelSelector.length > 0) {
        queryParamList.push(`listOptions.labelSelector=${labelSelector}`);
    }
    if (filter.resourceVersion) {
        queryParamList.push(`listOptions.resourceVersion=${filter.resourceVersion}`);
    }
    return queryParamList;
}

export function labelSelectorParams(phases?: Array<WorkflowPhase>, labels?: Array<string>): any {
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
