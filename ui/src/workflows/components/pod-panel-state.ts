import {NODE_PHASE, NodePhase, Pod} from '../../shared/models';

export type PodPanelState = 'shown' | 'deleted' | 'waiting' | 'not-found';

export function podPanelState(pod: Pod | undefined, deleted: boolean, nodePhase: NodePhase | undefined): PodPanelState {
    if (pod) {
        return deleted ? 'deleted' : 'shown';
    }
    return nodePhase === NODE_PHASE.PENDING || nodePhase === NODE_PHASE.RUNNING ? 'waiting' : 'not-found';
}
