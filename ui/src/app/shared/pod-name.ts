import {NodeStatus} from '../../models';

export const POD_NAME_V1 = 'v1';
export const POD_NAME_V2 = 'v2';

export const maxK8sResourceNameLength = 253;
export const k8sNamingHashLength = 10;

// getPodName returns a deterministic pod name
export const getPodName = (workflowName: string, nodeName: string, templateName: string, nodeID: string, version: string): string => {
    if (version === POD_NAME_V2) {
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

    return node.templateRef.template;
};
