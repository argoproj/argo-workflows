import {Observable, Observer} from 'rxjs';

import {catchError, map} from 'rxjs/operators';
import * as models from '../../../models';
import {Workflow, WorkflowList} from '../../../models';
import requests from './requests';
import {WorkflowDeleteResponse} from './responses';

export class WorkflowsService {
    public create(workflow: Workflow, namespace: string) {
        return requests
            .post(`api/v1/workflows/${namespace}`)
            .send({workflow})
            .then(res => res.body as Workflow);
    }

    public list(phases: string[], namespace: string) {
        return requests.get(`api/v1/workflows/${namespace}?${this.queryParams({phases}).join('&')}`).then(res => res.body as WorkflowList);
    }

    public get(namespace: string, name: string) {
        return requests.get(`api/v1/workflows/${namespace}/${name}`).then(res => res.body as Workflow);
    }

    public watch(filter: {namespace?: string; name?: string; phases?: Array<string>}): Observable<models.kubernetes.WatchEvent<Workflow>> {
        const url = `api/v1/workflow-events/${filter.namespace || ''}?${this.queryParams(filter).join('&')}`;

        return requests.loadEventSource(url, true).map(data => JSON.parse(data).result as models.kubernetes.WatchEvent<Workflow>);
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

    public terminate(name: string, namespace: string) {
        return requests.put(`api/v1/workflows/${namespace}/${name}/terminate`).then(res => res.body as Workflow);
    }

    public delete(name: string, namespace: string): Promise<WorkflowDeleteResponse> {
        return requests.delete(`api/v1/workflows/${namespace}/${name}`).then(res => res.body as WorkflowDeleteResponse);
    }

    public getContainerLogs(workflow: Workflow, nodeId: string, container: string, archived: boolean): Observable<string> {
        // we firstly try to get the logs from the API,
        // but if that fails, then we try and get them from the artifacts
        const logsFromArtifacts: Observable<string> = Observable.create((observer: Observer<string>) => {
            requests
                .get(this.getArtifactDownloadUrl(workflow, nodeId, container + '-logs', archived))
                .then(resp => {
                    resp.text.split('\n').forEach(line => observer.next(line));
                })
                .catch(err => observer.error(err));
            // tslint:disable-next-line
            return () => {
            };
        });
        return requests
            .loadEventSource(
                `api/v1/workflows/${workflow.metadata.namespace}/${workflow.metadata.name}/${nodeId}/log` +
                    `?logOptions.container=${container}&logOptions.tailLines=20&logOptions.follow=true`
            )
            .pipe(
                map(line => JSON.parse(line).result.content),
                catchError(() => logsFromArtifacts)
            );
    }

    public getArtifactDownloadUrl(workflow: Workflow, nodeId: string, artifactName: string, archived: boolean) {
        return archived
            ? `artifacts-by-uid/${workflow.metadata.uid}/${nodeId}/${encodeURIComponent(artifactName)}?Authorization=${localStorage.getItem('token')}`
            : `artifacts/${workflow.metadata.namespace}/${workflow.metadata.name}/${nodeId}/${encodeURIComponent(artifactName)}?Authorization=${localStorage.getItem('token')}`;
    }

    private queryParams(filter: {namespace?: string; name?: string; phases?: Array<string>}) {
        const queryParams: string[] = [];
        if (filter.name) {
            queryParams.push(`listOptions.fieldSelector=metadata.name=${filter.name}`);
        }
        if (filter.phases && filter.phases.length > 0) {
            queryParams.push(`listOptions.labelSelector=workflows.argoproj.io/phase in (${filter.phases.join(',')})`);
        }
        return queryParams;
    }
}
