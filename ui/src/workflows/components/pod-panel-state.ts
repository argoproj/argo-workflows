import {NODE_PHASE, NodePhase, Pod} from '../../shared/models';

export type PodPanelState = 'shown' | 'deleted' | 'waiting' | 'loading' | 'not-found';

export function podPanelState(pod: Pod | undefined, deleted: boolean, nodePhase: NodePhase | undefined, settled: boolean): PodPanelState {
    if (pod) {
        return deleted ? 'deleted' : 'shown';
    }
    if (nodePhase === NODE_PHASE.PENDING || nodePhase === NODE_PHASE.RUNNING) {
        return 'waiting';
    }
    return settled ? 'not-found' : 'loading';
}
