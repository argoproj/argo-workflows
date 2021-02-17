import {Observable} from 'rxjs';
import {CronWorkflow, CronWorkflowList} from '../../../models';
import * as models from '../../../models';
import {queryParams} from './common';
import requests from './requests';

export class CronWorkflowService {
    public create(cronWorkflow: CronWorkflow, namespace: string) {
        return requests
            .post(`api/v1/cron-workflows/${namespace}`)
            .send({cronWorkflow})
            .then(res => res.body as CronWorkflow);
    }

    public list(namespace: string) {
        return requests.get(`api/v1/cron-workflows/${namespace}`).then(res => res.body as CronWorkflowList);
    }

    public get(name: string, namespace: string) {
        return requests.get(`api/v1/cron-workflows/${namespace}/${name}`).then(res => res.body as CronWorkflow);
    }

    public update(cronWorkflow: CronWorkflow, name: string, namespace: string) {
        return requests
            .put(`api/v1/cron-workflows/${namespace}/${name}`)
            .send({cronWorkflow})
            .then(res => res.body as CronWorkflow);
    }

    public delete(name: string, namespace: string) {
        return requests.delete(`api/v1/cron-workflows/${namespace}/${name}`);
    }

    public suspend(name: string, namespace: string) {
        return requests.put(`api/v1/cron-workflows/${namespace}/${name}/suspend`).then(res => res.body as CronWorkflow);
    }

    public resume(name: string, namespace: string) {
        return requests.put(`api/v1/cron-workflows/${namespace}/${name}/resume`).then(res => res.body as CronWorkflow);
    }

    public watch(filter: {namespace?: string; name?: string; resourceVersion?: string}): Observable<models.kubernetes.WatchEvent<CronWorkflow>> {
        const url = `api/v1/cron-workflow-events/${filter.namespace || ''}?${queryParams(filter).join('&')}`;
        return requests.loadEventSource(url).map(data => data && (JSON.parse(data).result as models.kubernetes.WatchEvent<CronWorkflow>));
    }

    public watchFields(filter: {namespace?: string; name?: string; resourceVersion?: string}): Observable<models.kubernetes.WatchEvent<CronWorkflow>> {
        const params = queryParams(filter);
        const fields = [
            'result.object.metadata.name'
            // 'result.object.metadata.namespace',
            // 'result.object.metadata.resourceVersion',
            // 'result.object.metadata.creationTimestamp',
            // 'result.object.metadata.uid',
            // 'result.object.status.active',
            // 'result.object.status.lastScheduledTime',
            // 'result.type',
            // 'result.object.metadata.labels'
        ];
        params.push(`fields=${fields.join(',')}`);
        const url = `api/v1/cron-workflow-events/${filter.namespace || ''}?${params.join('&')}`;
        return requests.loadEventSource(url).map(data => data && (JSON.parse(data).result as models.kubernetes.WatchEvent<CronWorkflow>));
    }
}
