import * as models from '../../../models';
import {Pagination} from '../pagination';
import {Utils} from '../utils';
import requests from './requests';

export const WorkflowTemplateService = {
    create(template: models.WorkflowTemplate, namespace: string) {
        return requests
            .post(`api/v1/workflow-templates/${namespace}`)
            .send({template})
            .then(res => res.body as models.WorkflowTemplate);
    },

    list(namespace: string, labels?: string[], namePattern?: string, pagination?: Pagination) {
        return requests
            .get(`api/v1/workflow-templates/${namespace}?${Utils.queryParams({labels, namePattern, pagination}).join('&')}`)
            .then(res => res.body as models.WorkflowTemplateList);
    },

    get(name: string, namespace: string) {
        return requests.get(`api/v1/workflow-templates/${namespace}/${name}`).then(res => res.body as models.WorkflowTemplate);
    },

    update(template: models.WorkflowTemplate, name: string, namespace: string) {
        return requests
            .put(`api/v1/workflow-templates/${namespace}/${name}`)
            .send({template})
            .then(res => res.body as models.WorkflowTemplate);
    },

    delete(name: string, namespace: string) {
        return requests.delete(`api/v1/workflow-templates/${namespace}/${name}`);
    }
};
