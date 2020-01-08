import {CronWorkflow, CronWorkflowList} from '../../../models';
import requests from './requests';

export class CronWorkflowService {
    public list(namespace: string) {
        return requests
            .get(`api/v1/cron-workflows/${namespace}`)
            .then(res => res.body as CronWorkflowList)
            .then(list => list.items || []);
    }

    public update(template: CronWorkflow, templateName: string, namespace: string): Promise<CronWorkflow> {
        return requests
            .put(`api/v1/cron-workflows/${namespace}/${templateName}`)
            .send({
                templateName,
                namespace,
                template
            })
            .then(res => res.body as CronWorkflow);
    }

    public get(name: string, namespace: string): Promise<CronWorkflow> {
        return requests.get(`api/v1/cron-workflows/${namespace}/${name}`).then(res => res.body as CronWorkflow);
    }

    public delete(name: string, namespace: string): Promise<CronWorkflow> {
        return requests.delete(`api/v1/cron-workflows/${namespace}/${name}`).then(res => res.body as CronWorkflow);
    }

    public create(template: CronWorkflow): Promise<CronWorkflow> {
        return requests
            .post(`api/v1/cron-workflows`)
            .send({template})
            .then(res => res.body as CronWorkflow);
    }
}
