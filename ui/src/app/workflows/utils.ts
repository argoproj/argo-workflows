import {WorkflowsPagination} from '../../models';
import {Utils} from '../shared/utils';

export const WorkflowsUtils = {
    queryParams(filter: {
        namespace?: string;
        name?: string;
        namePrefix?: string;
        namePattern?: string;
        phases?: Array<string>;
        labels?: Array<string>;
        createdAfter?: Date;
        finishedBefore?: Date;
        pagination?: WorkflowsPagination;
        resourceVersion?: string;
    }) {
        const queryParams: string[] = [];
        const fieldSelector = Utils.fieldSelectorParams(filter.namespace, filter.name, filter.createdAfter, filter.finishedBefore);
        if (fieldSelector.length > 0) {
            queryParams.push(`listOptions.fieldSelector=${fieldSelector}`);
        }
        const labelSelector = Utils.labelSelectorParams(filter.phases, filter.labels);
        if (labelSelector.length > 0) {
            queryParams.push(`listOptions.labelSelector=${labelSelector}`);
        }
        if (filter.pagination) {
            if (filter.pagination) {
                queryParams.push(`paginationOptions.wfContinue=${filter.pagination.wfOffset}`);
                queryParams.push(`paginationOptions.archivedContinue=${filter.pagination.archivedOffset}`);
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
        if (filter.resourceVersion) {
            queryParams.push(`listOptions.resourceVersion=${filter.resourceVersion}`);
        }
        return queryParams;
    }
};
