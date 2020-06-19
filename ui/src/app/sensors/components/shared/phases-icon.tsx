import * as React from 'react';
import {NodePhase} from '../../model/sensors';

export const PhaseIcon = ({value}: {value: NodePhase}) => {
    let className = 'fa fa-clock status-icon--init';
    switch (value) {
        case 'Complete':
            className = 'fa fa-check-circle status-icon--success';
            break;
        case 'Active':
            className = 'fa fa-circle-notch fa-spin status-icon--running';
            break;
        case 'Error':
            className = 'fa fa-times-circle status-icon--failed';
            break;
    }
    return <i className={className} />;
};
