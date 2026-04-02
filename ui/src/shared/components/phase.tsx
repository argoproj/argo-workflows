import * as React from 'react';

import {PhaseIcon} from './phase-icon';

export function Phase({value}: {value: string}) {
    return (
        <span>
            <PhaseIcon value={value} /> {value}
        </span>
    );
}
