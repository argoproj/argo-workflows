import {Observable, Observer} from 'rxjs';

import {catchError, map} from 'rxjs/operators';
import * as models from '../../../models';
import requests from './requests';
import {WorkflowDeleteResponse} from './responses';

export class WorkflowsService {
    public get(namespace: string, name: string): Promise<models.Workflow> {
        return requests
            .get(`api/v1/workflows/${namespace}/${name}`)
            .then(res => res.body as models.Workflow)
            .then(this.populateDefaultFields);
    }

    public list(phases: string[], namespace: string): Promise<models.Workflow[]> {
        return requests
            .get(`api/v1/workflows/${namespace}`)
            .then(res => res.body as models.WorkflowList)
            .then()
            .then(list => (list.items || []).map(this.populateDefaultFields).filter(wf => phases.length === 0 || phases.includes(wf.status.phase)));
    }

    public watch(filter: {namespace?: string; name?: string; phases?: Array<string>}): Observable<models.kubernetes.WatchEvent<models.Workflow>> {
        const queryParams: string[] = [];
        if (filter.name) {
            queryParams.push(`listOptions.fieldSelector=metadata.name=${filter.name}`);
        }
        const url = `api/v1/workflow-events/${filter.namespace || ''}?${queryParams.join('&')}`;

        return requests
            .loadEventSource(url)
            .repeat()
            .retry()
            .map(data => JSON.parse(data).result as models.kubernetes.WatchEvent<models.Workflow>)
            .filter(wf => filter.phases === undefined || filter.phases.includes(wf.object.status.phase))
            .map(watchEvent => {
                watchEvent.object = this.populateDefaultFields(watchEvent.object);
                return watchEvent;
            });
    }

    public retry(workflowName: string, namespace: string): Promise<models.Workflow> {
        return requests
            .put(`api/v1/workflows/${namespace}/${workflowName}/retry`)
            .then(res => res.body as models.Workflow)
            .then(this.populateDefaultFields);
    }

    public resubmit(workflowName: string, namespace: string): Promise<models.Workflow> {
        return requests
            .put(`api/v1/workflows/${namespace}/${workflowName}/resubmit`)
            .then(res => res.body as models.Workflow)
            .then(this.populateDefaultFields);
    }

    public suspend(workflowName: string, namespace: string): Promise<models.Workflow> {
        return requests
            .put(`api/v1/workflows/${namespace}/${workflowName}/suspend`)
            .then(res => res.body as models.Workflow)
            .then(this.populateDefaultFields);
    }

    public resume(workflowName: string, namespace: string): Promise<models.Workflow> {
        return requests
            .put(`api/v1/workflows/${namespace}/${workflowName}/resume`)
            .then(res => res.body as models.Workflow)
            .then(this.populateDefaultFields);
    }

    public terminate(workflowName: string, namespace: string): Promise<models.Workflow> {
        return requests
            .put(`api/v1/workflows/${namespace}/${workflowName}/terminate`)
            .then(res => res.body as models.Workflow)
            .then(this.populateDefaultFields);
    }

    public delete(workflowName: string, namespace: string): Promise<WorkflowDeleteResponse> {
        return requests.delete(`api/v1/workflows/${namespace}/${workflowName}`).then(res => res.body as WorkflowDeleteResponse);
    }

    public create(workflow: models.Workflow, namespace: string): Promise<models.Workflow> {
        return requests
            .post(`api/v1/workflows/${namespace}`)
            .send({
                namespace,
                workflow
            })
            .then(res => res.body as models.Workflow)
            .then(this.populateDefaultFields);
    }

    public getContainerLogs(workflow: models.Workflow, nodeId: string, container: string, archived: boolean): Observable<string> {
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
                    `?logOptions.container=${container}&logOptions.tailLines=20&logOptions.follow=true&logOptions.timestamps=true`
            )
            .pipe(
                map(line => JSON.parse(line).result.content),
                catchError(() => logsFromArtifacts)
            );
    }

    public getArtifactDownloadUrl(workflow: models.Workflow, nodeId: string, artifactName: string, archived: boolean) {
        return archived
            ? `/artifacts-by-uid/${workflow.metadata.namespace}/${workflow.metadata.uid}/${nodeId}/${encodeURIComponent(artifactName)}?Authorization=${localStorage.getItem(
                  'token'
              )}`
            : `/artifacts/${workflow.metadata.namespace}/${workflow.metadata.name}/${nodeId}/${encodeURIComponent(artifactName)}?Authorization=${localStorage.getItem('token')}`;
    }

    private populateDefaultFields(workflow: models.Workflow): models.Workflow {
        workflow = {status: {nodes: {}}, ...workflow} as models.Workflow;
        workflow.status.nodes = workflow.status.nodes || {};
        return workflow;
    }
}
