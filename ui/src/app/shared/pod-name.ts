import {NodeStatus, Workflow} from '../../models';
import {ANNOTATION_KEY_POD_NAME_VERSION} from './annotations';

export const POD_NAME_V1 = 'v1';
export const POD_NAME_V2 = 'v2';

export const maxK8sResourceNameLength = 253;
export const k8sNamingHashLength = 10;

// getPodName returns a deterministic pod name
// In case templateName is not defined or that version is explicitly set to  POD_NAME_V1, it will return the nodeID (v1)
// In other cases it will return a combination of workflow name, template name, and a hash (v2)
// note: this is intended to be equivalent to the server-side Go code in workflow/util/pod_name.go
export function getPodName(workflow: Workflow, node: NodeStatus): string {
    const workflowName = workflow.metadata.name;
    const version = workflow.metadata?.annotations?.[ANNOTATION_KEY_POD_NAME_VERSION];
    const templateName = getTemplateNameFromNode(node);

    if (version !== POD_NAME_V1 && templateName !== '') {
        if (workflowName === node.name) {
            return workflowName;
        }

        const prefix = ensurePodNamePrefixLength(`${workflowName}-${templateName}`);
        const hash = createFNVHash(node.name);
        return `${prefix}-${hash}`;
    }

    return node.id;
};

function ensurePodNamePrefixLength(prefix: string): string {
    const maxPrefixLength = maxK8sResourceNameLength - k8sNamingHashLength;

    if (prefix.length > maxPrefixLength - 1) {
        return prefix.substring(0, maxPrefixLength - 1);
    }

    return prefix;
}

function createFNVHash(input: string): number {
    let hashint = 2166136261;

    for (let i = 0; i < input.length; i++) {
        const character = input.charCodeAt(i);
        hashint = hashint ^ character;
        hashint += (hashint << 1) + (hashint << 4) + (hashint << 7) + (hashint << 8) + (hashint << 24);
    }

    return hashint >>> 0;
}

function getTemplateNameFromNode(node: NodeStatus): string {
    if (node.templateName && node.templateName !== '') {
        return node.templateName;
    }

    // fall back to v1 pod names if no templateName or templateRef defined
    if (node?.templateRef === undefined || node?.templateRef.template === '') {
        return '';
    }

    return node.templateRef.template;
}
