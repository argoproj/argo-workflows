import * as moment from 'moment';
import * as React from 'react';
import Moment from 'react-moment';
import {NODE_PHASE, NodePhase} from '../../../models';
import {formatDuration} from '../duration';
import {ProgressLine} from './progress-line';

// duration panel in seconds
export const DurationPanel = (props: {phase: NodePhase; duration: number; estimatedDuration?: number}) => {
    if (props.phase === NODE_PHASE.RUNNING && props.estimatedDuration) {
        return (
            <>
                <span title={'Estimate duration: ' + formatDuration(props.estimatedDuration)}>
                    <ProgressLine progress={props.duration / props.estimatedDuration} width={32} height={8} />
                </span>{' '}
                {formatDuration(props.duration)}
            </>
        );
    }
    return <>{formatDuration(props.duration)}</>;
};

export const DurationFromNow = ({getDate, frequency = 1000}: {getDate: () => string; frequency?: number}) => {
    const [now, setNow] = React.useState(moment());
    const [date, setDate] = React.useState(getDate);
    React.useEffect(() => {
        const interval = setInterval(() => {
            setNow(moment());
            setDate(getDate);
        }, frequency);
        return () => {
            clearInterval(interval);
        };
    }, []);

    return <Moment duration={now} date={date} format='dd:hh:mm:ss' />;
};
