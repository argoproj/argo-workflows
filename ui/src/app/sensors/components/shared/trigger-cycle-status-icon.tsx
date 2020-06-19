import * as React from 'react';

export const TriggerCycleStatusIcon = ({value}: {value: string}) => {
    let className = 'fa fa-clock status-icon--init';
    switch (value) {
        case 'Success':
            className = 'fa fa-check-circle status-icon--success';
            break;
        case 'Failure':
            className = 'fa fa-times-circle status-icon--failed';
            break;
    }
    return <i className={className} />;
};
