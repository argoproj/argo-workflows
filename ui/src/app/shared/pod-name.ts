import {NodeStatus} from '../../models';

export const POD_NAME_V1 = 'v1';
export const POD_NAME_V2 = 'v2';

export const maxK8sResourceNameLength = 253;
export const k8sNamingHashLength = 10;

// getPodName returns a deterministic pod name. It returns a combination of the
// workflow name, template name, and a hash if the POD_NAME_V2 annotation is
// set. If the templateName or templateRef is not defined on a given node, it
// falls back to POD_NAME_V1
export const getPodName = (workflowName: string, nodeName: string, templateName: string, nodeID: string, version: string): string => {
    if (version === POD_NAME_V2 && templateName !== '') {
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
    const data = Buffer.from(input);

    let hashint = 2166136261;

    /* tslint:disable:no-bitwise */
    for (const character of data) {
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
    if (!node.templateRef) {
        return '';
    }

    return node.templateRef.template;
};
