import * as models from '../../models';
import {NODE_PHASE} from '../../models';
import {Pagination} from './pagination';

const managedNamespaceKey = 'managedNamespace';
const currentNamespaceKey = 'current_namespace';

export const Utils = {
    statusIconClasses(status: string): string {
        let classes = [];
        switch (status) {
            case NODE_PHASE.ERROR:
            case NODE_PHASE.FAILED:
                classes = ['fa-times-circle', 'status-icon--failed'];
                break;
            case NODE_PHASE.SUCCEEDED:
                classes = ['fa-check-circle', 'status-icon--success'];
                break;
            case NODE_PHASE.RUNNING:
                classes = ['fa-circle-notch', 'status-icon--running', 'status-icon--spin'];
                break;
            case NODE_PHASE.PENDING:
                classes = ['fa-clock', 'status-icon--pending', 'status-icon--slow-spin'];
                break;
            default:
                classes = ['fa-clock', 'status-icon--init'];
                break;
        }
        return classes.join(' ');
    },

    shortNodeName(node: {name: string; displayName: string}): string {
        return node.displayName || node.name;
    },

    isWorkflowSuspended(wf: models.Workflow): boolean {
        if (!wf || !wf.spec) {
            return false;
        }
        if (wf.spec.suspend !== undefined && wf.spec.suspend) {
            return true;
        }
        if (wf.status && wf.status.nodes) {
            for (const node of Object.values(wf.status.nodes)) {
                if (node.type === 'Suspend' && node.phase === 'Running') {
                    return true;
                }
            }
        }
        return false;
    },

    isWorkflowRunning(wf: models.Workflow): boolean {
        if (!wf || !wf.spec) {
            return false;
        }
        return wf.status.phase === 'Running';
    },

    set managedNamespace(value: string) {
        if (value) {
            localStorage.setItem(managedNamespaceKey, value);
        } else {
            localStorage.removeItem(managedNamespaceKey);
        }
    },

    get managedNamespace() {
        return this.fixLocalStorageString(localStorage.getItem(managedNamespaceKey));
    },

    fixLocalStorageString(x: string): string {
        // empty string is valid, so we cannot use `truthy`
        if (x !== null && x !== 'null' && x !== 'undefined') {
            return x;
        }
    },

    onNamespaceChange(value: string) {
        // noop
    },

    set currentNamespace(value: string) {
        if (value != null) {
            localStorage.setItem(currentNamespaceKey, value);
        } else {
            localStorage.removeItem(currentNamespaceKey);
        }
        this.onNamespaceChange(this.currentNamespace);
    },

    get currentNamespace() {
        // we always prefer the managed namespace
        return this.managedNamespace || this.fixLocalStorageString(localStorage.getItem(currentNamespaceKey));
    },

    // return a namespace, favoring managed namespace when set
    getNamespace(namespace: string) {
        return this.managedNamespace || namespace;
    },

    // return a namespace, never return null/undefined, defaults to "default"
    getNamespaceWithDefault(namespace: string) {
        return this.managedNamespace || namespace || this.currentNamespace || 'default';
    },

    queryParams(filter: {
        namespace?: string;
        name?: string;
        namePrefix?: string;
        phases?: Array<string>;
        labels?: Array<string>;
        minStartedAt?: Date;
        maxStartedAt?: Date;
        pagination?: Pagination;
        resourceVersion?: string;
    }) {
        const queryParams: string[] = [];
        const fieldSelector = this.fieldSelectorParams(filter.namespace, filter.name, filter.minStartedAt, filter.maxStartedAt);
        if (fieldSelector.length > 0) {
            queryParams.push(`listOptions.fieldSelector=${fieldSelector}`);
        }
        const labelSelector = this.labelSelectorParams(filter.phases, filter.labels);
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
        if (filter.resourceVersion) {
            queryParams.push(`listOptions.resourceVersion=${filter.resourceVersion}`);
        }
        return queryParams;
    },

    fieldSelectorParams(namespace?: string, name?: string, minStartedAt?: Date, maxStartedAt?: Date) {
        let fieldSelector = '';
        if (namespace) {
            fieldSelector += 'metadata.namespace=' + namespace + ',';
        }
        if (name) {
            fieldSelector += 'metadata.name=' + name + ',';
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
    },

    labelSelectorParams(phases?: Array<string>, labels?: Array<string>) {
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
};
