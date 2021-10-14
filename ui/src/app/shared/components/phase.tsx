import * as React from 'react';
import {PhaseIcon} from './phase-icon';

export const Phase = ({value}: {value: string}) => {
    return (
        <>
            <PhaseIcon value={value} /> {value}
        </>
    );
};
