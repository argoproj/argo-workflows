import * as models from '../../../models';
import {Utils} from '../utils';
import requests from './requests';

export class WorkflowTemplateService {
    public create(template: models.WorkflowTemplate, namespace: string) {
        return requests
            .post(`api/v1/workflow-templates/${namespace}`)
            .send({template})
            .then(res => res.body as models.WorkflowTemplate);
    }

    public list(namespace: string, labels: string[]) {
        return requests
            .get(`api/v1/workflow-templates/${namespace}?${Utils.queryParams({labels}).join('&')}`)
            .then(res => res.body as models.WorkflowTemplateList)
            .then(list => list.items || []);
    }

    public get(name: string, namespace: string) {
        return requests.get(`api/v1/workflow-templates/${namespace}/${name}`).then(res => res.body as models.WorkflowTemplate);
    }

    public update(template: models.WorkflowTemplate, name: string, namespace: string) {
        return requests
            .put(`api/v1/workflow-templates/${namespace}/${name}`)
            .send({template})
            .then(res => res.body as models.WorkflowTemplate);
    }

    public delete(name: string, namespace: string) {
        return requests.delete(`api/v1/workflow-templates/${namespace}/${name}`);
    }
}
