import {CronWorkflow, CronWorkflowList} from '../../../models';
import requests from './requests';

export class CronWorkflowService {
    public create(cronWorkflow: CronWorkflow, namespace: string) {
        return requests
            .post(`api/v1/cron-workflows/${namespace}`)
            .send({cronWorkflow})
            .then(res => res.body as CronWorkflow);
    }

    public list(namespace: string, labels: string[] = []) {
        return requests
            .get(`api/v1/cron-workflows/${namespace}?${this.queryParams({labels}).join('&')}`)
            .then(res => res.body as CronWorkflowList)
            .then(list => list.items || []);
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

    private queryParams(filter: {labels?: Array<string>}) {
        const queryParams: string[] = [];
        const labelSelector = this.labelSelectorParams(filter.labels);
        if (labelSelector.length > 0) {
            queryParams.push(`listOptions.labelSelector=${labelSelector}`);
        }

        return queryParams;
    }

    private labelSelectorParams(labels?: Array<string>) {
        let labelSelector = '';
        if (labels && labels.length > 0) {
            if (labelSelector.length > 0) {
                labelSelector += ',';
            }
            labelSelector += labels.join(',');
        }
        return labelSelector;
    }
}
