import {Observable, Observer} from 'rxjs';

import {catchError, map} from 'rxjs/operators';
import * as models from '../../../models';
import requests from './requests';
import {WorkflowDeleteResponse} from './responses';

export class WorkflowsService {
    public get(namespace: string, name: string): Promise<models.Workflow> {
        return requests
            .get(`/api/v1/workflows/${namespace}/${name}`)
            .then(res => res.body as models.Workflow)
            .then(this.populateDefaultFields);
    }

    public list(phases: string[], namespace: string): Promise<models.Workflow[]> {
        return requests
            .get(`/api/v1/workflows/${namespace}`)
            .query({phase: phases})
            .then(res => res.body as models.WorkflowList)
            .then(list => (list.items || []).map(this.populateDefaultFields));
    }

    public watch(filter?: {namespace: string; name: string} | Array<string>): Observable<models.kubernetes.WatchEvent<models.Workflow>> {
        let url = '/api/v1/workflow-events/';
        if (filter) {
            if (filter instanceof Array) {
                const phases = (filter as Array<string>).map(phase => `listOptions.fieldSelector=status.phase=${phase}`).join('&');
                url = `${url}?${phases}`;
            } else {
                const workflow = filter as {namespace: string; name: string};
                url = `${url}${workflow.namespace}?listOptions.fieldSelector=metadata.name=${workflow.name}`;
            }
        }
        return requests
            .loadEventSource(url)
            .repeat()
            .retry()
            .map(data => JSON.parse(data).result as models.kubernetes.WatchEvent<models.Workflow>)
            .map(watchEvent => {
                watchEvent.object = this.populateDefaultFields(watchEvent.object);
                return watchEvent;
            });
    }

    public retry(workflowName: string, namespace: string): Promise<models.Workflow> {
        return requests
            .put(`/api/v1/workflows/${namespace}/${workflowName}/retry`)
            .then(res => res.body as models.Workflow)
            .then(this.populateDefaultFields);
    }

    public resubmit(workflowName: string, namespace: string): Promise<models.Workflow> {
        return requests
            .put(`/api/v1/workflows/${namespace}/${workflowName}/resubmit`)
            .then(res => res.body as models.Workflow)
            .then(this.populateDefaultFields);
    }

    public suspend(workflowName: string, namespace: string): Promise<models.Workflow> {
        return requests
            .put(`/api/v1/workflows/${namespace}/${workflowName}/suspend`)
            .then(res => res.body as models.Workflow)
            .then(this.populateDefaultFields);
    }

    public resume(workflowName: string, namespace: string): Promise<models.Workflow> {
        return requests
            .put(`/api/v1/workflows/${namespace}/${workflowName}/resume`)
            .then(res => res.body as models.Workflow)
            .then(this.populateDefaultFields);
    }

    public terminate(workflowName: string, namespace: string): Promise<models.Workflow> {
        return requests
            .put(`/api/v1/workflows/${namespace}/${workflowName}/terminate`)
            .then(res => res.body as models.Workflow)
            .then(this.populateDefaultFields);
    }

    public delete(workflowName: string, namespace: string): Promise<WorkflowDeleteResponse> {
        return requests.delete(`/api/v1/workflows/${namespace}/${workflowName}`).then(res => res.body as WorkflowDeleteResponse);
    }

    public create(workflow: models.Workflow, namespace: string): Promise<models.Workflow> {
        return requests
            .post(`/api/v1/workflows/${namespace}`)
            .send({
                namespace,
                workflow
            })
            .then(res => res.body as models.Workflow)
            .then(this.populateDefaultFields);
    }

    public getContainerLogs(workflow: models.Workflow, nodeId: string, container: string, historical: boolean): Observable<string> {
        // we firstly try to get the logs from the API,
        // but if that fails, then we try and get them from the artifacts
        const logsFromArtifacts: Observable<string> = Observable.create((observer: Observer<string>) => {
            requests
                .get(this.getArtifactDownloadUrl(workflow, nodeId, container + '-logs', historical))
                .then(resp => {
                    resp.text.split('\n').forEach(line => observer.next(line));
                })
                .catch(err => observer.error(err));
            // tslint:disable-next-line
            return () => {
            };
        });
        return requests.loadEventSource(`/api/v1/workflows/${workflow.metadata.namespace}/${workflow.metadata.name}/${nodeId}/log?logOptions.container=${container}`).pipe(
            map(line => JSON.parse(line).result.content),
            catchError(() => logsFromArtifacts)
        );
    }

    public getArtifactDownloadUrl(workflow: models.Workflow, nodeId: string, artifactName: string, historical: boolean) {
        return historical
            ? `/historical-artifacts/${workflow.metadata.namespace}/${workflow.metadata.uid}/${nodeId}/${encodeURIComponent(artifactName)}?Authorization=${localStorage.getItem(
                  'token'
              )}`
            : `/artifacts/${workflow.metadata.namespace}/${workflow.metadata.name}/${nodeId}/${encodeURIComponent(artifactName)}?Authorization=${localStorage.getItem('token')}`;
    }

    private populateDefaultFields(workflow: models.Workflow): models.Workflow {
        workflow = {status: {nodes: {}}, ...workflow};
        workflow.status.nodes = workflow.status.nodes || {};
        return workflow;
    }
}
