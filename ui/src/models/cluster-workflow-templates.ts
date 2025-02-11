import * as kubernetes from 'argo-ui/src/models/kubernetes';
import {WorkflowTemplateSpec} from './workflow-templates';

export interface ClusterWorkflowTemplate {
    apiVersion?: string;
    kind?: string;
    metadata: kubernetes.ObjectMeta;
    spec: WorkflowTemplateSpec;
}

export interface ClusterWorkflowTemplateList {
    apiVersion?: string;
    kind?: string;
    metadata: kubernetes.ListMeta;
    items: ClusterWorkflowTemplate[];
}
