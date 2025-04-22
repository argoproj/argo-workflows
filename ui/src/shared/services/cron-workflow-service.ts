import {CronWorkflow, CronWorkflowList} from '../models';
import requests from './requests';
import {queryParams} from './utils';

export const CronWorkflowService = {
    create(cronWorkflow: CronWorkflow, namespace: string) {
        return requests
            .post(`api/v1/cron-workflows/${namespace}`)
            .send({cronWorkflow})
            .then(res => res.body as CronWorkflow);
    },

    list(namespace: string, labels: string[] = []) {
        return requests
            .get(`api/v1/cron-workflows/${namespace}?${queryParams({labels}).join('&')}`)
            .then(res => res.body as CronWorkflowList)
            .then(list => list.items || []);
    },

    get(name: string, namespace: string) {
        return requests.get(`api/v1/cron-workflows/${namespace}/${name}`).then(res => res.body as CronWorkflow);
    },

    update(cronWorkflow: CronWorkflow, name: string, namespace: string) {
        return requests
            .put(`api/v1/cron-workflows/${namespace}/${name}`)
            .send({cronWorkflow})
            .then(res => res.body as CronWorkflow);
    },

    delete(name: string, namespace: string) {
        return requests.delete(`api/v1/cron-workflows/${namespace}/${name}`);
    },

    suspend(name: string, namespace: string) {
        return requests.put(`api/v1/cron-workflows/${namespace}/${name}/suspend`).then(res => res.body as CronWorkflow);
    },

    resume(name: string, namespace: string) {
        return requests.put(`api/v1/cron-workflows/${namespace}/${name}/resume`).then(res => res.body as CronWorkflow);
    }
};
