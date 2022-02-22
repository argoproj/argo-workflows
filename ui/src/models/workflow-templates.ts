import * as kubernetes from 'argo-ui/src/models/kubernetes';
import {WorkflowSpec} from './workflows';

export interface WorkflowTemplate {
    apiVersion?: string;
    kind?: string;
    metadata: kubernetes.ObjectMeta;
    spec: WorkflowTemplateSpec;
}

export interface WorkflowTemplateSpec extends WorkflowSpec {
    workflowMetadata?: kubernetes.ObjectMeta;
}

export interface WorkflowTemplateList {
    apiVersion?: string;
    kind?: string;
    metadata: kubernetes.ListMeta;
    items: WorkflowTemplate[];
}
