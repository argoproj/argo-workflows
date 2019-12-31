import * as kubernetes from 'argo-ui/src/models/kubernetes';
import {Arguments, Template} from './workflows';

export interface WorkflowTemplate {
    apiVersion?: string;
    kind?: string;
    metadata: kubernetes.ObjectMeta;
    spec: WorkflowTemplateSpec;
}

export interface WorkflowTemplateSpec {
    /**
     * Arguments contain the parameters and artifacts sent to the workflow entrypoint.
     * Parameters are referencable globally using the 'workflow' variable prefix. e.g. {{workflow.parameters.myparam}}
     */
    arguments?: Arguments;
    /**
     * Templates is a list of workflow templates used in a workflow
     */
    templates: Template[];
}

export interface WorkflowTemplateList {
    apiVersion?: string;
    kind?: string;
    metadata: kubernetes.ListMeta;
    items: WorkflowTemplate[];
}
