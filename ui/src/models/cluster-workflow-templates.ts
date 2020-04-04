import * as kubernetes from 'argo-ui/src/models/kubernetes';
import {WorkflowSpec} from './workflows';

export interface ClusterWorkflowTemplate {
    apiVersion?: string;
    kind?: string;
    metadata: kubernetes.ObjectMeta;
    spec: WorkflowSpec;
}

export interface ClusterWorkflowTemplateList {
    apiVersion?: string;
    kind?: string;
    metadata: kubernetes.ListMeta;
    items: ClusterWorkflowTemplate[];
}
