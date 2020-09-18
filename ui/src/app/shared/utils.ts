import {Observable} from 'rxjs';
import * as models from '../../models';
import {NODE_PHASE} from '../../models';

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

    toObservable<T>(val: T | Observable<T> | Promise<T>): Observable<T> {
        const observable = val as Observable<T>;
        if (observable && observable.subscribe && observable.catch) {
            return observable as Observable<T>;
        }
        return Observable.from([val as T]);
    },

    tryJsonParse(input: string) {
        try {
            return (input && JSON.parse(input)) || null;
        } catch {
            return null;
        }
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

    onNamespaceChange(value: string) {
        // noop
    },

    setCurrentNamespace(value: string): void {
        if (value) {
            localStorage.setItem('current_namespace', value);
            this.onNamespaceChange(value);
        }
    },

    getCurrentNamespace(): string {
        return localStorage.getItem('current_namespace');
    }
};
