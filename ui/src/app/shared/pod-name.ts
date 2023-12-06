import {NodeStatus} from '../../models';

export const POD_NAME_V1 = 'v1';
export const POD_NAME_V2 = 'v2';

export const maxK8sResourceNameLength = 253;
export const k8sNamingHashLength = 10;

// getPodName returns a deterministic pod name
// In case templateName is not defined or that version is explicitly set to  POD_NAME_V1, it will return the nodeID (v1)
// In other cases it will return a combination of workflow name, template name, and a hash (v2)
// note: this is intended to be equivalent to the server-side Go code in workflow/util/pod_name.go
export const getPodName = (workflowName: string, nodeName: string, templateName: string, nodeID: string, version: string): string => {
    if (version !== POD_NAME_V1 && templateName !== '') {
        if (workflowName === nodeName) {
            return workflowName;
        }

        const prefix = ensurePodNamePrefixLength(`${workflowName}-${templateName}`);

        const hash = createFNVHash(nodeName);
        return `${prefix}-${hash}`;
    }

    return nodeID;
};

export const ensurePodNamePrefixLength = (prefix: string): string => {
    const maxPrefixLength = maxK8sResourceNameLength - k8sNamingHashLength;

    if (prefix.length > maxPrefixLength - 1) {
        return prefix.substring(0, maxPrefixLength - 1);
    }

    return prefix;
};

export const createFNVHash = (input: string): number => {
    let hashint = 2166136261;

    for (let i = 0; i < input.length; i++) {
        const character = input.charCodeAt(i);
        hashint = hashint ^ character;
        hashint += (hashint << 1) + (hashint << 4) + (hashint << 7) + (hashint << 8) + (hashint << 24);
    }

    return hashint >>> 0;
};

export const getTemplateNameFromNode = (node: NodeStatus): string => {
    if (node.templateName && node.templateName !== '') {
        return node.templateName;
    }

    // fall back to v1 pod names if no templateName or templateRef defined
    if (node?.templateRef === undefined || node?.templateRef.template === '') {
        return '';
    }

    return node.templateRef.template;
};
