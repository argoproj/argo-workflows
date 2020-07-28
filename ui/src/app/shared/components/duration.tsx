import * as React from 'react';
import {NodePhase} from '../../../models';
import {formatDuration} from '../duration';

// duration panel in milliseconds
export const DurationPanel = (props: {phase: NodePhase; duration: number; estimatedDuration?: number}) => {
    // (props.phase == 'Running' || props.phase == 'Pending') &&
    if (props.estimatedDuration && props.estimatedDuration > 0) {
        return (
            <>
                {formatDuration(props.duration / 1000)} ETA {formatDuration(props.estimatedDuration / 1000)} ({((100 * props.duration) / props.estimatedDuration).toFixed()}%)
            </>
        );
    }
    return <>{formatDuration(props.duration / 1000)}</>;
};
