import {Observable} from 'rxjs';

import * as models from '../../../models';
import requests from './requests';
import {WorkflowDeleteResponse} from './responses';

export class WorkflowsService {
    public get(namespace: string, name: string): Promise<models.Workflow> {
        return requests
            .get(`/workflows/${namespace}/${name}`)
            .then(res => res.body as models.Workflow)
            .then(this.populateDefaultFields);
    }

    public list(phases: string[], namespace: string): Promise<models.Workflow[]> {
        return requests
            .get(`/workflows/${namespace}`)
            .query({phase: phases})
            .then(res => res.body as models.WorkflowList)
            .then(list => list.items.map(this.populateDefaultFields));
    }

    public watch(filter?: {namespace: string; name: string} | Array<string>): Observable<models.kubernetes.WatchEvent<models.Workflow>> {
        let url = '/workflows/live';
        if (filter) {
            if (filter instanceof Array) {
                const phases = (filter as Array<string>).map(phase => `phase=${phase}`).join('&');
                url = `${url}?${phases}`;
            } else {
                const workflow = filter as {namespace: string; name: string};
                url = `${url}?namespace=${workflow.namespace}&name=${workflow.name}`;
            }
        }
        return requests
            .loadEventSource(url)
            .repeat()
            .retry()
            .map(data => JSON.parse(data) as models.kubernetes.WatchEvent<models.Workflow>)
            .map(watchEvent => {
                watchEvent.object = this.populateDefaultFields(watchEvent.object);
                return watchEvent;
            });
    }

    public retry(workflowName: string, namespace: string): Promise<models.Workflow> {
        return requests
            .put(`/workflows/${namespace}/${workflowName}/retry`)
            .then(res => res.body as models.Workflow)
            .then(this.populateDefaultFields);
    }

    public resubmit(workflowName: string, namespace: string): Promise<models.Workflow> {
        return requests
            .put(`/workflows/${namespace}/${workflowName}/resubmit`)
            .then(res => res.body as models.Workflow)
            .then(this.populateDefaultFields);
    }

    public suspend(workflowName: string, namespace: string): Promise<models.Workflow> {
        return requests
            .put(`/workflows/${namespace}/${workflowName}/suspend`)
            .then(res => res.body as models.Workflow)
            .then(this.populateDefaultFields);
    }

    public resume(workflowName: string, namespace: string): Promise<models.Workflow> {
        return requests
            .put(`/workflows/${namespace}/${workflowName}/resume`)
            .then(res => res.body as models.Workflow)
            .then(this.populateDefaultFields);
    }

    public terminate(workflowName: string, namespace: string): Promise<models.Workflow> {
        return requests
            .put(`/workflows/${namespace}/${workflowName}/terminate`)
            .then(res => res.body as models.Workflow)
            .then(this.populateDefaultFields);
    }

    public delete(workflowName: string, namespace: string): Promise<WorkflowDeleteResponse> {
        return requests.delete(`/workflows/${namespace}/${workflowName}`).then(res => res.body as WorkflowDeleteResponse);
    }

    public getContainerLogs(workflow: models.Workflow, nodeId: string, container: string): Observable<string> {
        return requests.loadEventSource(`/logs/${workflow.metadata.namespace}/${workflow.metadata.name}/${nodeId}/${container}`).map(line => {
            return line ? line + '\n' : line;
        });
    }

    public getArtifactDownloadUrl(workflow: models.Workflow, nodeId: string, artifactName: string) {
        return `/api/workflows/${workflow.metadata.namespace}/${workflow.metadata.name}/artifacts/${nodeId}/${encodeURIComponent(artifactName)}`;
    }

    private populateDefaultFields(workflow: models.Workflow): models.Workflow {
        workflow = {status: {nodes: {}}, ...workflow};
        workflow.status.nodes = workflow.status.nodes || {};
        return workflow;
    }
}
