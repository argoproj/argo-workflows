import * as models from '../../../models';
import requests from './requests';
import {WorkflowTemplate} from "../../../models";

export class WorkflowTemplateService {
    public list(namespace: string) {
        return requests
            .get(`/workflowtemplates/${namespace}`)
            .then(res => res.body as models.WorkflowTemplateList)
            .then(list => list.items || []);
    }

    public update(template: models.WorkflowTemplate, templateName: string, namespace: string): Promise<WorkflowTemplate> {
        return requests
            .put(`/workflowtemplates/${namespace}/${templateName}`)
            .send({
                templateName,
                namespace,
                template
            })
            .then(res => res.body as models.WorkflowTemplate);
    }

    public get(name: string, namespace: string): Promise<WorkflowTemplate> {
        return requests.get(`/workflowtemplates/${namespace}/${name}`).then(res => res.body as models.WorkflowTemplate);
    }

    public delete(name: string, namespace: string): Promise<WorkflowTemplate> {
        return requests.delete(`/workflowtemplates/${namespace}/${name}`).then(res => res.body as models.WorkflowTemplate);
    }
}
