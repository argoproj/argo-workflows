import * as React from 'react';
import {NODE_PHASE, NodePhase} from '../../../models';
import {formatDuration} from '../duration';

// duration panel in milliseconds
export const DurationPanel = (props: {phase: NodePhase; duration: number; estimatedDuration?: number}) => {
    if ((props.phase === NODE_PHASE.PENDING || props.phase === NODE_PHASE.RUNNING) && props.estimatedDuration && props.estimatedDuration > 0) {
        return (
            <>
                <span title={'Estimate duration: ' + formatDuration(props.estimatedDuration / 1000)}>
                    <svg width={32} height={8}>
                        <rect width={32} height={8} rx={4} fill='gray' />
                        <rect x={2} y={2} width={(32 - 4) * Math.min(1, props.duration / props.estimatedDuration)} height={4} fill='white' rx={2} />
                    </svg>
                </span>{' '}
                {formatDuration(props.duration / 1000)}
            </>
        );
    }
    return <>{formatDuration(props.duration / 1000)}</>;
};
