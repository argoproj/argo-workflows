import * as models from '../../../models';
import {CronWorkflow} from '../../../models';
import requests from './requests';

export class CronWorkflowService {
    public list(namespace: string) {
        return requests
            .get(`api/v1/cron-wrokflows/${namespace}`)
            .then(res => res.body as models.CronWorkflowList)
            .then(list => list.items || []);
    }

    public update(template: models.CronWorkflow, templateName: string, namespace: string): Promise<CronWorkflow> {
        return requests
            .put(`api/v1/cron-wrokflows/${namespace}/${templateName}`)
            .send({
                templateName,
                namespace,
                template
            })
            .then(res => res.body as models.CronWorkflow);
    }

    public get(name: string, namespace: string): Promise<CronWorkflow> {
        return requests.get(`api/v1/cron-wrokflows/${namespace}/${name}`).then(res => res.body as models.CronWorkflow);
    }

    public delete(name: string, namespace: string): Promise<CronWorkflow> {
        return requests.delete(`api/v1/cron-wrokflows/${namespace}/${name}`).then(res => res.body as models.CronWorkflow);
    }

    public create(template: models.CronWorkflow, namespace: string): Promise<models.CronWorkflow> {
        return requests
            .post(`api/v1/cron-wrokflows`)
            .send({
                template,
                namespace
            })
            .then(res => res.body as models.CronWorkflow);
    }
}
