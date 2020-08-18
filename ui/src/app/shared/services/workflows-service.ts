import {Observable, Observer} from 'rxjs';

import {catchError, map} from 'rxjs/operators';
import * as models from '../../../models';
import {Event, Workflow, WorkflowList} from '../../../models';
import {SubmitOpts} from '../../../models/submit-opts';
import {Pagination} from '../pagination';
import requests from './requests';
import {WorkflowDeleteResponse} from './responses';

export class WorkflowsService {
    public create(workflow: Workflow, namespace: string) {
        return requests
            .post(`api/v1/workflows/${namespace}`)
            .send({workflow})
            .then(res => res.body as Workflow);
    }

    public list(namespace: string, phases: string[], labels: string[], pagination: Pagination) {
        const params = this.queryParams({phases, labels});
        if (pagination.offset) {
            params.push(`listOptions.continue=${pagination.offset}`);
        }
        if (pagination.limit) {
            params.push(`listOptions.limit=${pagination.limit}`);
        }
        const fields = [
            'metadata',
            'items.metadata.uid',
            'items.metadata.name',
            'items.metadata.namespace',
            'items.metadata.labels',
            'items.status.phase',
            'items.status.finishedAt',
            'items.status.startedAt',
            'items.spec.suspend'
        ];
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

    public getContainerLogs(workflow: Workflow, nodeId: string, container: string, archived: boolean): Observable<string> {
        // we firstly try to get the logs from the API,
        // but if that fails, then we try and get them from the artifacts
        const logsFromArtifacts: Observable<string> = Observable.create((observer: Observer<string>) => {
            requests
                .get(this.getArtifactLogsUrl(workflow, nodeId, container, archived))
                .then(resp => {
                    resp.text.split('\n').forEach(line => observer.next(line));
                })
                .catch(err => observer.error(err));
            // tslint:disable-next-line
            return () => {};
        });
        return requests
            .loadEventSource(
                `api/v1/workflows/${workflow.metadata.namespace}/${workflow.metadata.name}/${nodeId}/log` + `?logOptions.container=${container}&logOptions.follow=true`
            )
            .pipe(
                map(line => JSON.parse(line).result.content),
                catchError(() => logsFromArtifacts)
            );
    }

    public getArtifactLogsUrl(workflow: Workflow, nodeId: string, container: string, archived: boolean) {
        return this.getArtifactDownloadUrl(workflow, nodeId, container + '-logs', archived);
    }

    public getArtifactDownloadUrl(workflow: Workflow, nodeId: string, artifactName: string, archived: boolean) {
        return archived
            ? `artifacts-by-uid/${workflow.metadata.uid}/${nodeId}/${encodeURIComponent(artifactName)}`
            : `artifacts/${workflow.metadata.namespace}/${workflow.metadata.name}/${nodeId}/${encodeURIComponent(artifactName)}`;
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
