export const maxK8sResourceNameLength = 253;
export const k8sNamingHashLength = 10;

// getPodName returns a deterministic pod name
export const getPodName = (workflowName: string, nodeName: string, templateName: string, nodeID: string): string => {
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
