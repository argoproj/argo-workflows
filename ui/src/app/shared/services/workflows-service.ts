import {Observable} from 'rxjs';
import * as models from '../../../models';
import {Event, NodeStatus, Workflow, WorkflowList} from '../../../models';
import {SubmitOpts} from '../../../models/submit-opts';
import {Pagination} from '../pagination';
import requests from './requests';
import {WorkflowDeleteResponse} from './responses';

function isString(value: any): value is string {
    return typeof value === 'string';
}

export class WorkflowsService {
    public create(workflow: Workflow, namespace: string) {
        return requests
            .post(`api/v1/workflows/${namespace}`)
            .send({workflow})
            .then(res => res.body as Workflow);
    }

    public list(
        namespace: string,
        phases: string[],
        labels: string[],
        pagination: Pagination,
        fields = [
            'metadata',
            'items.metadata.uid',
            'items.metadata.name',
            'items.metadata.namespace',
            'items.metadata.labels',
            'items.status.phase',
            'items.status.finishedAt',
            'items.status.startedAt',
            'items.status.estimatedDuration',
            'items.status.progress',
            'items.spec.suspend'
        ]
    ) {
        const params = this.queryParams({phases, labels});
        if (pagination.offset) {
            params.push(`listOptions.continue=${pagination.offset}`);
        }
        if (pagination.limit) {
            params.push(`listOptions.limit=${pagination.limit}`);
        }
        params.push(`fields=${fields.join(',')}`);
        return requests.get(`api/v1/workflows/${namespace}?${params.join('&')}`).then(res => res.body as WorkflowList);
    }

    public get(namespace: string, name: string) {
        return requests.get(`api/v1/workflows/${namespace}/${name}`).then(res => res.body as Workflow);
    }

    public watch(filter: {
        namespace?: string;
        name?: string;
        phases?: Array<string>;
        labels?: Array<string>;
        resourceVersion?: string;
    }): Observable<models.kubernetes.WatchEvent<Workflow>> {
        const url = `api/v1/workflow-events/${filter.namespace || ''}?${this.queryParams(filter).join('&')}`;
        return requests.loadEventSource(url).map(data => JSON.parse(data).result as models.kubernetes.WatchEvent<Workflow>);
    }

    public watchEvents(namespace: string, fieldSelector: string): Observable<Event> {
        return requests.loadEventSource(`api/v1/stream/events/${namespace}?listOptions.fieldSelector=${fieldSelector}`).map(data => JSON.parse(data).result as Event);
    }

    public watchFields(filter: {
        namespace?: string;
        name?: string;
        phases?: Array<string>;
        labels?: Array<string>;
        resourceVersion?: string;
    }): Observable<models.kubernetes.WatchEvent<Workflow>> {
        const params = this.queryParams(filter);
        const fields = [
            'result.object.metadata.name',
            'result.object.metadata.namespace',
            'result.object.metadata.resourceVersion',
            'result.object.metadata.uid',
            'result.object.status.finishedAt',
            'result.object.status.phase',
            'result.object.status.startedAt',
            'result.object.status.estimatedDuration',
            'result.object.status.progress',
            'result.type',
            'result.object.metadata.labels',
            'result.object.spec.suspend'
        ];
        params.push(`fields=${fields.join(',')}`);
        const url = `api/v1/workflow-events/${filter.namespace || ''}?${params.join('&')}`;
        return requests.loadEventSource(url).map(data => JSON.parse(data).result as models.kubernetes.WatchEvent<Workflow>);
    }

    public retry(name: string, namespace: string) {
        return requests.put(`api/v1/workflows/${namespace}/${name}/retry`).then(res => res.body as Workflow);
    }

    public resubmit(name: string, namespace: string) {
        return requests.put(`api/v1/workflows/${namespace}/${name}/resubmit`).then(res => res.body as Workflow);
    }

    public suspend(name: string, namespace: string) {
        return requests.put(`api/v1/workflows/${namespace}/${name}/suspend`).then(res => res.body as Workflow);
    }

    public resume(name: string, namespace: string) {
        return requests.put(`api/v1/workflows/${namespace}/${name}/resume`).then(res => res.body as Workflow);
    }

    public stop(name: string, namespace: string) {
        return requests.put(`api/v1/workflows/${namespace}/${name}/stop`).then(res => res.body as Workflow);
    }

    public terminate(name: string, namespace: string) {
        return requests.put(`api/v1/workflows/${namespace}/${name}/terminate`).then(res => res.body as Workflow);
    }

    public delete(name: string, namespace: string): Promise<WorkflowDeleteResponse> {
        return requests.delete(`api/v1/workflows/${namespace}/${name}`).then(res => res.body as WorkflowDeleteResponse);
    }

    public submit(kind: string, name: string, namespace: string, submitOptions?: SubmitOpts) {
        return requests
            .post(`api/v1/workflows/${namespace}/submit`)
            .send({namespace, resourceKind: kind, resourceName: name, submitOptions})
            .then(res => res.body as Workflow);
    }

    public getContainerLogsFromCluster(workflow: Workflow, nodeId: string, container: string): Observable<string> {
        const podLogsURL = `api/v1/workflows/${workflow.metadata.namespace}/${workflow.metadata.name}/${nodeId}/log?logOptions.container=${container}&logOptions.follow=true`;
        return requests
            .loadEventSource(podLogsURL)
            .map(line => JSON.parse(line).result.content)
            .filter(isString)
            .catch(() => {
                // When an error occurs on an observable, RxJS is hard-coded to unsubscribe from the stream.  In the case
                // that the connection to the server was interrupted while the node is still pending or running, this is not
                // correct since we actually want the EventSource to re-connect and continue streaming logs.  In the event
                // that the pod has completed, then we want to allow the unsubscribe to happen since no additional logs exist.
                return Observable.fromPromise(this.isWorkflowNodePendingOrRunning(workflow, nodeId)).switchMap(isPendingOrRunning => {
                    if (isPendingOrRunning) {
                        return this.getContainerLogsFromCluster(workflow, nodeId, container);
                    }

                    // If our workflow is completed, then simply complete the Observable since nothing else
                    // should be omitted
                    return Observable.empty();
                });
            });
    }

    public async isWorkflowNodePendingOrRunning(workflow: Workflow, nodeId: string) {
        // We always refresh the workflow rather than inspecting the state locally since it doubles
        // as a check to determine whether or not the API is currently reachable
        const updatedWorkflow = await this.get(workflow.metadata.namespace, workflow.metadata.name);
        return this.isNodePendingOrRunning(updatedWorkflow.status.nodes[nodeId]);
    }

    public getContainerLogsFromArtifact(workflow: Workflow, nodeId: string, container: string, archived: boolean) {
        return Observable.of(this.hasArtifactLogs(workflow, nodeId, container))
            .switchMap(hasArtifactLogs => {
                if (!hasArtifactLogs) {
                    throw new Error('no artifact logs are available');
                }

                return Observable.fromPromise(requests.get(this.getArtifactLogsUrl(workflow, nodeId, container, archived)));
            })
            .mergeMap(r => r.text.split('\n'));
    }

    public getContainerLogs(workflow: Workflow, nodeId: string, container: string, archived: boolean): Observable<string> {
        const getLogsFromArtifact = () => this.getContainerLogsFromArtifact(workflow, nodeId, container, archived);

        // If our workflow is archived, don't even bother inspecting the cluster for logs since it's likely
        // that the Workflow and associated pods have been deleted
        if (archived) {
            return getLogsFromArtifact();
        }

        return this.getContainerLogsFromCluster(workflow, nodeId, container).catch(getLogsFromArtifact);
    }

    public getArtifactLogsUrl(workflow: Workflow, nodeId: string, container: string, archived: boolean) {
        return this.getArtifactDownloadUrl(workflow, nodeId, container + '-logs', archived);
    }

    public getArtifactDownloadUrl(workflow: Workflow, nodeId: string, artifactName: string, archived: boolean) {
        return archived
            ? `artifacts-by-uid/${workflow.metadata.uid}/${nodeId}/${encodeURIComponent(artifactName)}`
            : `artifacts/${workflow.metadata.namespace}/${workflow.metadata.name}/${nodeId}/${encodeURIComponent(artifactName)}`;
    }

    private isNodePendingOrRunning(node: NodeStatus) {
        return node.phase === models.NODE_PHASE.PENDING || node.phase === models.NODE_PHASE.RUNNING;
    }

    private hasArtifactLogs(workflow: Workflow, nodeId: string, container: string) {
        const node = workflow.status.nodes[nodeId];

        if (!node || !node.outputs) {
            return false;
        }

        return node.outputs.artifacts.findIndex(a => a.name === `${container}-logs`) !== -1;
    }

    private queryParams(filter: {namespace?: string; name?: string; phases?: Array<string>; labels?: Array<string>; resourceVersion?: string}) {
        const queryParams: string[] = [];
        if (filter.name) {
            queryParams.push(`listOptions.fieldSelector=metadata.name=${filter.name}`);
        }
        const labelSelector = this.labelSelectorParams(filter.phases, filter.labels);
        if (labelSelector.length > 0) {
            queryParams.push(`listOptions.labelSelector=${labelSelector}`);
        }
        if (filter.resourceVersion) {
            queryParams.push(`listOptions.resourceVersion=${filter.resourceVersion}`);
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
