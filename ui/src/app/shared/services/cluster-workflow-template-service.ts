import * as models from '../../../models';
import requests from './requests';

export class ClusterWorkflowTemplateService {
    public create(template: models.ClusterWorkflowTemplate) {
        return requests
            .post(`api/v1/cluster-workflow-templates`)
            .send({template})
            .then(res => res.body as models.ClusterWorkflowTemplate);
    }

    public list() {
        return requests
            .get(`api/v1/cluster-workflow-templates`)
            .then(res => res.body as models.ClusterWorkflowTemplateList)
            .then(list => list.items || []);
    }

    public get(name: string) {
        return requests.get(`api/v1/cluster-workflow-templates/${name}`).then(res => res.body as models.ClusterWorkflowTemplate);
    }

    public update(template: models.ClusterWorkflowTemplate, name: string) {
        return requests
            .put(`api/v1/cluster-workflow-templates/${name}`)
            .send({template})
            .then(res => res.body as models.ClusterWorkflowTemplate);
    }

    public delete(name: string) {
        return requests.delete(`api/v1/cluster-workflow-templates/${name}`);
    }
}
