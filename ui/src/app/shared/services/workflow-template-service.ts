import * as models from '../../../models';
import requests from './requests';

export class WorkflowTemplateService {
    public list(namespace: string) {
        return requests
            .get(`/workflowtemplates/${namespace}`)
            .then(res => res.body as models.WorkflowTemplateList)
            .then(list => list.items || []);
    }

    public get(name: string, namespace: string) {
        return requests.get(`/workflowtemplates/${namespace}/${name}`).then(res => res.body as models.WorkflowTemplate);
    }

    public delete(name: string, namespace: string) {
        return requests.delete(`/workflowtemplates/${namespace}/${name}`).then(res => res.body as models.WorkflowTemplate);
    }
}
