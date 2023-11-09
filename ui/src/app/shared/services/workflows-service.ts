import {EMPTY, from, Observable, of} from 'rxjs';
import {catchError, filter, map, mergeMap, switchMap} from 'rxjs/operators';
import * as models from '../../../models';
import {Event, LogEntry, NodeStatus, Workflow, WorkflowList, WorkflowPhase} from '../../../models';
import {ResubmitOpts, RetryOpts} from '../../../models';
import {SubmitOpts} from '../../../models/submit-opts';
import {uiUrl} from '../base';
import {Pagination} from '../pagination';
import {Utils} from '../utils';
import requests from './requests';
import {WorkflowDeleteResponse} from './responses';

function isString(value: any): value is string {
    return typeof value === 'string';
}

function isNodePendingOrRunning(node: NodeStatus) {
    return node.phase === models.NODE_PHASE.PENDING || node.phase === models.NODE_PHASE.RUNNING;
}

function hasArtifactLogs(workflow: Workflow, nodeId: string, container: string) {
    const node = workflow.status.nodes[nodeId];

    if (!node || !node.outputs || !node.outputs.artifacts) {
        return false;
    }

    return node.outputs.artifacts.findIndex(a => a.name === `${container}-logs`) !== -1;
}

export const WorkflowsService = {
    create(workflow: Workflow, namespace: string) {
        return requests
            .post(`api/v1/workflows/${namespace}`)
            .send({workflow})
            .then(res => res.body as Workflow);
    },

    list(
        namespace: string,
        phases: WorkflowPhase[],
        labels: string[],
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
        const params = Utils.queryParams({phases, labels, pagination});
        params.push(`fields=${fields.join(',')}`);
        return requests.get(`api/v1/workflows/${namespace}?${params.join('&')}`).then(res => res.body as WorkflowList);
    },

    get(namespace: string, name: string) {
        return requests.get(`api/v1/workflows/${namespace}/${name}`).then(res => res.body as Workflow);
    },

    getArchived(namespace: string, uid: string) {
        return requests.get(`api/v1/archived-workflows/${uid}?namespace=${namespace}`).then(res => res.body as models.Workflow);
    },

    watch(query: {
        namespace?: string;
        name?: string;
        phases?: Array<WorkflowPhase>;
        labels?: Array<string>;
        resourceVersion?: string;
    }): Observable<models.kubernetes.WatchEvent<Workflow>> {
        const url = `api/v1/workflow-events/${query.namespace || ''}?${Utils.queryParams(query).join('&')}`;
        return requests.loadEventSource(url).pipe(map(data => data && (JSON.parse(data).result as models.kubernetes.WatchEvent<Workflow>)));
    },

    watchEvents(namespace: string, fieldSelector: string): Observable<Event> {
        return requests
            .loadEventSource(`api/v1/stream/events/${namespace}?listOptions.fieldSelector=${fieldSelector}`)
            .pipe(map(data => data && (JSON.parse(data).result as Event)));
    },

    watchFields(query: {
        namespace?: string;
        name?: string;
        phases?: Array<WorkflowPhase>;
        labels?: Array<string>;
        resourceVersion?: string;
    }): Observable<models.kubernetes.WatchEvent<Workflow>> {
        const params = Utils.queryParams(query);
        const fields = [
            'result.object.metadata.name',
            'result.object.metadata.namespace',
            'result.object.metadata.resourceVersion',
            'result.object.metadata.creationTimestamp',
            'result.object.metadata.uid',
            'result.object.status.finishedAt',
            'result.object.status.phase',
            'result.object.status.message',
            'result.object.status.startedAt',
            'result.object.status.estimatedDuration',
            'result.object.status.progress',
            'result.type',
            'result.object.metadata.labels',
            'result.object.metadata.annotations',
            'result.object.spec.suspend'
        ];
        params.push(`fields=${fields.join(',')}`);
        const url = `api/v1/workflow-events/${query.namespace || ''}?${params.join('&')}`;
        return requests.loadEventSource(url).pipe(map(data => data && (JSON.parse(data).result as models.kubernetes.WatchEvent<Workflow>)));
    },

    retry(name: string, namespace: string, opts?: RetryOpts) {
        return requests
            .put(`api/v1/workflows/${namespace}/${name}/retry`)
            .send(opts)
            .then(res => res.body as Workflow);
    },

    retryArchived(uid: string, namespace: string, opts?: RetryOpts) {
        return requests
            .put(`api/v1/archived-workflows/${uid}/retry`)
            .send({namespace, ...opts})
            .then(res => res.body as Workflow);
    },

    resubmit(name: string, namespace: string, opts?: ResubmitOpts) {
        return requests
            .put(`api/v1/workflows/${namespace}/${name}/resubmit`)
            .send(opts)
            .then(res => res.body as Workflow);
    },

    resubmitArchived(uid: string, namespace: string, opts?: ResubmitOpts) {
        return requests
            .put(`api/v1/archived-workflows/${uid}/resubmit`)
            .send({namespace, ...opts})
            .then(res => res.body as Workflow);
    },

    suspend(name: string, namespace: string) {
        return requests.put(`api/v1/workflows/${namespace}/${name}/suspend`).then(res => res.body as Workflow);
    },

    set(name: string, namespace: string, nodeFieldSelector: string, outputParameters: string) {
        return requests
            .put(`api/v1/workflows/${namespace}/${name}/set`)
            .send({nodeFieldSelector, outputParameters})
            .then(res => res.body as Workflow);
    },

    resume(name: string, namespace: string, nodeFieldSelector: string) {
        return requests
            .put(`api/v1/workflows/${namespace}/${name}/resume`)
            .send({nodeFieldSelector})
            .then(res => res.body as Workflow);
    },

    stop(name: string, namespace: string) {
        return requests.put(`api/v1/workflows/${namespace}/${name}/stop`).then(res => res.body as Workflow);
    },

    terminate(name: string, namespace: string) {
        return requests.put(`api/v1/workflows/${namespace}/${name}/terminate`).then(res => res.body as Workflow);
    },

    delete(name: string, namespace: string): Promise<WorkflowDeleteResponse> {
        return requests.delete(`api/v1/workflows/${namespace}/${name}`).then(res => res.body as WorkflowDeleteResponse);
    },

    deleteArchived(uid: string, namespace: string): Promise<WorkflowDeleteResponse> {
        if (namespace === '') {
            return requests.delete(`api/v1/archived-workflows/${uid}`).then(res => res.body as WorkflowDeleteResponse);
        } else {
            return requests.delete(`api/v1/archived-workflows/${uid}?namespace=${namespace}`).then(res => res.body as WorkflowDeleteResponse);
        }
    },

    submit(kind: string, name: string, namespace: string, submitOptions?: SubmitOpts) {
        return requests
            .post(`api/v1/workflows/${namespace}/submit`)
            .send({namespace, resourceKind: kind, resourceName: name, submitOptions})
            .then(res => res.body as Workflow);
    },

    getContainerLogsFromCluster(workflow: Workflow, podName: string, container: string, grep: string): Observable<LogEntry> {
        const namespace = workflow.metadata.namespace;
        const name = workflow.metadata.name;
        const podLogsURL = uiUrl(
            `api/v1/workflows/${namespace}/${name}/log?logOptions.container=${container}&grep=${grep}&logOptions.follow=true${podName ? `&podName=${podName}` : ''}`
        );
        return requests.loadEventSource(podLogsURL).pipe(
            filter(line => !!line),
            map(line => JSON.parse(line).result as LogEntry),
            filter(e => isString(e.content)),
            catchError(() => {
                // When an error occurs on an observable, RxJS is hard-coded to unsubscribe from the stream.  In the case
                // that the connection to the server was interrupted while the node is still pending or running, this is not
                // correct since we actually want the EventSource to re-connect and continue streaming logs.  In the event
                // that the pod has completed, then we want to allow the unsubscribe to happen since no additional logs exist.
                return from(this.isWorkflowNodePendingOrRunning(workflow, podName)).pipe(
                    switchMap(isPendingOrRunning => {
                        if (isPendingOrRunning) {
                            return this.getContainerLogsFromCluster(workflow, podName, container, grep);
                        }

                        // If our workflow is completed, then simply complete the Observable since nothing else
                        // should be omitted
                        return EMPTY;
                    })
                );
            })
        );
    },

    async isWorkflowNodePendingOrRunning(workflow: Workflow, nodeId?: string) {
        // We always refresh the workflow rather than inspecting the state locally since it doubles
        // as a check to determine whether or not the API is currently reachable
        const updatedWorkflow = await this.get(workflow.metadata.namespace, workflow.metadata.name);
        const node = updatedWorkflow.status.nodes[nodeId];
        if (!node) {
            return !updatedWorkflow.status || ['Pending', 'Running'].includes(updatedWorkflow.status.phase);
        }
        return isNodePendingOrRunning(node);
    },

    getContainerLogsFromArtifact(workflow: Workflow, nodeId: string, container: string, grep: string, archived: boolean): Observable<LogEntry> {
        return of(hasArtifactLogs(workflow, nodeId, container)).pipe(
            switchMap(isArtifactLogs => {
                if (!isArtifactLogs) {
                    if (!nodeId) {
                        throw new Error('Should specify a node when we get archived logs');
                    }
                    throw new Error('no artifact logs are available');
                }

                return from(requests.get(this.getArtifactLogsPath(workflow, nodeId, container, archived)));
            }),
            mergeMap(r => r.text.split('\n')),
            map(content => ({content, podName: workflow.status.nodes[nodeId].displayName} as LogEntry)),
            filter(x => !!x.content.match(grep))
        );
    },

    getContainerLogs(workflow: Workflow, podName: string, nodeId: string, container: string, grep: string, archived: boolean): Observable<LogEntry> {
        const getLogsFromArtifact = () => this.getContainerLogsFromArtifact(workflow, nodeId, container, grep, archived);
        const getLogsFromCluster = () => this.getContainerLogsFromCluster(workflow, podName, container, grep);

        // If our workflow is archived, don't even bother inspecting the cluster for logs since it's likely
        // that the Workflow and associated pods have been deleted
        if (archived) {
            return getLogsFromArtifact();
        }

        // return archived log if main container is finished and has artifact
        return from(this.isWorkflowNodePendingOrRunning(workflow, nodeId)).pipe(
            switchMap(isPendingOrRunning => {
                if (!isPendingOrRunning && hasArtifactLogs(workflow, nodeId, container) && container === 'main') {
                    return getLogsFromArtifact().pipe(catchError(getLogsFromCluster));
                }
                return getLogsFromCluster().pipe(catchError(getLogsFromArtifact));
            })
        );
    },

    getArtifactLogsPath(workflow: Workflow, nodeId: string, container: string, archived: boolean) {
        return this.artifactPath(workflow, nodeId, container + '-logs', archived, false);
    },

    getArtifactDownloadUrl(workflow: Workflow, nodeId: string, artifactName: string, archived: boolean, isInput: boolean) {
        return uiUrl(this.artifactPath(workflow, nodeId, artifactName, archived, isInput));
    },

    artifactPath(workflow: Workflow, nodeId: string, artifactName: string, archived: boolean, isInput: boolean) {
        if (!isInput) {
            return `artifact-files/${workflow.metadata.namespace}/${archived ? 'archived-workflows' : 'workflows'}/${
                archived ? workflow.metadata.uid : workflow.metadata.name
            }/${nodeId}/outputs/${artifactName}`;
        } else if (archived) {
            return `input-artifacts-by-uid/${workflow.metadata.uid}/${nodeId}/${encodeURIComponent(artifactName)}`;
        } else {
            return `input-artifacts/${workflow.metadata.namespace}/${workflow.metadata.name}/${nodeId}/${encodeURIComponent(artifactName)}`;
        }
    }
};
