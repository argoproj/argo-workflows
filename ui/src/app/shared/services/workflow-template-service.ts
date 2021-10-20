import * as models from '../../../models';
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
            .get(`api/v1/workflow-templates/${namespace}?${this.queryParams({labels}).join('&')}`)
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
