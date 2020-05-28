import * as React from 'react';
import {NodePhase} from '../../../models';
import {PhaseIcon} from './phase-icon';

export const Phase = ({value}: {value: NodePhase}) => {
    return (
        <>
            <PhaseIcon value={value} /> {value}
        </>
    );
};
