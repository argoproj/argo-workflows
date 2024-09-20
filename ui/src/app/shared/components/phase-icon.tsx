import classNames from 'classnames';
import * as React from 'react';

import {NODE_PHASE} from '../../../models';

export function statusIconClasses(status: string): string {
    let classes = [];
    switch (status) {
        case NODE_PHASE.ERROR:
        case NODE_PHASE.FAILED:
            classes = ['fa-times-circle', 'status-icon--failed'];
            break;
        case NODE_PHASE.SUCCEEDED:
            classes = ['fa-check-circle', 'status-icon--success'];
            break;
        case NODE_PHASE.RUNNING:
            classes = ['fa-circle-notch', 'status-icon--running', 'status-icon--spin'];
            break;
        case NODE_PHASE.PENDING:
            classes = ['fa-clock', 'status-icon--pending', 'status-icon--slow-spin'];
            break;
        default:
            classes = ['fa-clock', 'status-icon--init'];
            break;
    }
    return classes.join(' ');
}

export function PhaseIcon({value}: {value: string}) {
    return <i className={classNames('fa', statusIconClasses(value))} aria-hidden='true' />;
}
