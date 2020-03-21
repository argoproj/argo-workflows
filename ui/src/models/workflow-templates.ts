import * as kubernetes from 'argo-ui/src/models/kubernetes';
import {WorkflowSpec} from './workflows';

export interface WorkflowTemplate {
    apiVersion?: string;
    kind?: string;
    metadata: kubernetes.ObjectMeta;
    spec: WorkflowSpec;
}

export interface WorkflowTemplateList {
    apiVersion?: string;
    kind?: string;
    metadata: kubernetes.ListMeta;
    items: WorkflowTemplate[];
}
