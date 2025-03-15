import {NameFilterKeys} from '../../workflows/components/workflow-filters/workflow-filters';
import {Pagination} from '../pagination';

export function queryParams(filter: {
    namespace?: string;
    name?: string;
    namePrefix?: string;
    namePattern?: string;
    nameFilter?: NameFilterKeys;
    phases?: Array<string>;
    labels?: Array<string>;
    createdAfter?: Date;
    finishedBefore?: Date;
    pagination?: Pagination;
    resourceVersion?: string;
}) {
    const queryParams: string[] = [];
    const fieldSelector = fieldSelectorParams(filter.namespace, filter.name);
    if (fieldSelector.length > 0) {
        queryParams.push(`listOptions.fieldSelector=${fieldSelector}`);
    }
    const labelSelector = labelSelectorParams(filter.phases, filter.labels);
    if (labelSelector.length > 0) {
        queryParams.push(`listOptions.labelSelector=${labelSelector}`);
    }
    if (filter.pagination) {
        if (filter.pagination.offset) {
            queryParams.push(`listOptions.continue=${filter.pagination.offset}`);
        }
        if (filter.pagination.limit) {
            queryParams.push(`listOptions.limit=${filter.pagination.limit}`);
        }
    }
    if (filter.namePrefix) {
        queryParams.push(`namePrefix=${filter.namePrefix}`);
    }
    if (filter.namePattern) {
        queryParams.push(`namePattern=${filter.namePattern}`);
    }
    if (filter.nameFilter) {
        queryParams.push(`nameFilter=${filter.nameFilter}`);
    }
    if (filter.resourceVersion) {
        queryParams.push(`listOptions.resourceVersion=${filter.resourceVersion}`);
    }
    if (filter.createdAfter) {
        queryParams.push(`createdAfter=${filter.createdAfter.toISOString()}`);
    }
    if (filter.finishedBefore) {
        queryParams.push(`finishedBefore=${filter.finishedBefore.toISOString()}`);
    }
    return queryParams;
}

function fieldSelectorParams(namespace?: string, name?: string) {
    let fieldSelector = '';
    if (namespace) {
        fieldSelector += 'metadata.namespace=' + namespace + ',';
    }
    if (name) {
        fieldSelector += 'metadata.name=' + name + ',';
    }
    if (fieldSelector.endsWith(',')) {
        fieldSelector = fieldSelector.substring(0, fieldSelector.length - 1);
    }
    return fieldSelector;
}

function labelSelectorParams(phases?: Array<string>, labels?: Array<string>) {
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
