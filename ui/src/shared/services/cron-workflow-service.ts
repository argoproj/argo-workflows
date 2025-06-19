import {CronWorkflow, CronWorkflowList} from '../models';
import requests from './requests';
import {queryParams} from './utils';

// Handle CronWorkflows using the deprecated "schedule" field by automatically
// migrating them to use "schedules".
// Also, gracefully handle invalid CronWorkflows that are missing both
// "schedule" and "schedules".
function normalizeSchedules(cronWorkflow: any): CronWorkflow {
    cronWorkflow.spec.schedules ??= [];
    // TODO: Delete this once we drop support for "schedule"
    if ((cronWorkflow.spec.schedule ?? '') != '') {
        cronWorkflow.spec.schedules.push(cronWorkflow.spec.schedule);
        delete cronWorkflow.spec.schedule;
    }

    // Ensure when property is properly handled
    cronWorkflow.spec.when ??= [];
    // If when is a string in the API response, convert it to an array
    if (typeof cronWorkflow.spec.when === 'string') {
        cronWorkflow.spec.when = [cronWorkflow.spec.when];
    }

    return cronWorkflow as CronWorkflow;
}

export const CronWorkflowService = {
    create(cronWorkflow: CronWorkflow, namespace: string) {
        return requests
            .post(`api/v1/cron-workflows/${namespace}`)
            .send({cronWorkflow})
            .then(res => normalizeSchedules(res.body));
    },

    list(namespace: string, labels: string[] = []) {
        return requests
            .get(`api/v1/cron-workflows/${namespace}?${queryParams({labels}).join('&')}`)
            .then(res => res.body as CronWorkflowList)
            .then(list => (list.items || []).map(normalizeSchedules));
    },

    get(name: string, namespace: string) {
        return requests.get(`api/v1/cron-workflows/${namespace}/${name}`).then(res => normalizeSchedules(res.body));
    },

    update(cronWorkflow: CronWorkflow, name: string, namespace: string) {
        return requests
            .put(`api/v1/cron-workflows/${namespace}/${name}`)
            .send({cronWorkflow})
            .then(res => normalizeSchedules(res.body));
    },

    delete(name: string, namespace: string) {
        return requests.delete(`api/v1/cron-workflows/${namespace}/${name}`);
    },

    suspend(name: string, namespace: string) {
        return requests.put(`api/v1/cron-workflows/${namespace}/${name}/suspend`).then(res => normalizeSchedules(res.body));
    },

    resume(name: string, namespace: string) {
        return requests.put(`api/v1/cron-workflows/${namespace}/${name}/resume`).then(res => normalizeSchedules(res.body));
    }
};
