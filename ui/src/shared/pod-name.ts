import {ANNOTATION_KEY_POD_NAME_VERSION} from './annotations';
import {NodeStatus, PooledTaskCondition, PooledTaskState, PooledTaskStatus, Workflow} from './models';

export const POD_NAME_V1 = 'v1';
export const POD_NAME_V2 = 'v2';

export const maxK8sResourceNameLength = 253;
export const k8sNamingHashLength = 10;
const maxPrefixLength = maxK8sResourceNameLength - k8sNamingHashLength;

// getPodName returns a deterministic pod name
// In case templateName is not defined or that version is explicitly set to  POD_NAME_V1, it will return the nodeID (v1)
// In other cases it will return a combination of workflow name, template name, and a hash (v2)
// note: this is intended to be equivalent to the server-side Go code in workflow/util/pod_name.go#GeneratePodName
export function getPodName(workflow: Workflow, node: NodeStatus): string {
    const version = workflow.metadata?.annotations?.[ANNOTATION_KEY_POD_NAME_VERSION];
    if (version === POD_NAME_V1) {
        return node.id;
    }

    const workflowName = workflow.metadata.name;
    // a containerSet container node runs in the pod of the node it is bounded by
    // this part is from workflow/controller/container_set_template.go#executeContainerSet; the inverse never happens in the back-end, so is unique to the front-end
    const podNodeId = (node.type == 'Container' ? node.boundaryID : node.id) ?? node.id;
    if (workflowName === podNodeId) {
        return workflowName;
    }

    const templateName = getTemplateNameFromNode(node);
    let prefix = workflowName;
    if (templateName) {
        prefix += `-${templateName}`;
    }
    prefix = ensurePodNamePrefixLength(prefix);

    // the node ID is the workflow name followed by the hash of the node name (with any collision suffix),
    // so take the hash from there rather than rehashing the node name, which would put colliding nodes in the same pod
    const idPrefix = `${workflowName}-`;
    const hash = `${podNodeId?.startsWith(idPrefix) ? podNodeId.substring(idPrefix.length) : createFNVHash(node.name)}`;
    // a widened 64-bit node ID can exceed the hash-length budget ensurePodNamePrefixLength reserves; shorten the prefix, not the hash
    const excess = prefix.length + 1 + hash.length - maxK8sResourceNameLength;
    if (excess > 0) {
        prefix = prefix.substring(0, prefix.length - excess);
    }
    return `${prefix}-${hash}`;
}

function findCondition(task: PooledTaskStatus, type: PooledTaskCondition['type']): PooledTaskCondition | undefined {
    return (task.conditions || []).find(c => c.type === type);
}

// getPooledTaskState returns where a pooled task is in its hand-off, from its conditions:
// Interrupted once the controller has interrupted it, Claimed while a worker holds it, and Enqueued otherwise
// note: this is intended to be equivalent to the server-side Go code in pkg/apis/workflow/v1alpha1/pool_types.go#State
export function getPooledTaskState(task: PooledTaskStatus): PooledTaskState {
    const isTrue = (type: PooledTaskCondition['type']) => findCondition(task, type)?.status === 'True';
    if (isTrue('Interrupted')) {
        return 'Interrupted';
    }
    return isTrue('Claimed') ? 'Claimed' : 'Enqueued';
}

export type PooledTaskMarker = 'queued' | 'unclaimed' | 'released';

// getPooledTaskPodName returns the name a pooled node shows where a pod node shows its pod name:
// the claiming worker pod once a worker has claimed the task, otherwise the task's `{{node.taskName}}`
// with a marker saying why no worker holds it: queued while the node runs, then unclaimed or released once it has ended
// note: this is intended to be equivalent to the server-side Go code in workflow/util/pod_name.go#PooledTaskPodName
export function getPooledTaskPodName(task: PooledTaskStatus, ended: boolean): {name: string; marker?: PooledTaskMarker} {
    const state = getPooledTaskState(task);
    if (state !== 'Enqueued' && task.workerPodName) {
        return {name: task.workerPodName};
    }
    if (state !== 'Enqueued') {
        return {name: task.taskName};
    }
    if (!ended) {
        return {name: task.taskName, marker: 'queued'};
    }
    const claimed = findCondition(task, 'Claimed');
    const released = claimed?.status === 'False' && claimed.reason === 'Released';
    return {name: task.taskName, marker: released ? 'released' : 'unclaimed'};
}

export function ensurePodNamePrefixLength(prefix: string): string {
    if (prefix.length > maxPrefixLength - 1) {
        return prefix.substring(0, maxPrefixLength - 1);
    }

    return prefix;
}

export function createFNVHash(input: string): number {
    let hashint = 2166136261;

    const encoder = new TextEncoder();
    const bytes = encoder.encode(input);

    for (let i = 0; i < bytes.length; i++) {
        const byte = bytes[i];
        hashint = hashint ^ byte;
        hashint += (hashint << 1) + (hashint << 4) + (hashint << 7) + (hashint << 8) + (hashint << 24);
    }

    return hashint >>> 0;
}

export function getTemplateNameFromNode(node: NodeStatus): string {
    // fall back to v1 pod names if no templateName or templateRef defined
    return node.templateName || node.templateRef?.template || '';
}
