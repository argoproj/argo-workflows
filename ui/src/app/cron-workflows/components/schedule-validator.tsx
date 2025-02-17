import x from 'cronstrue';
import * as React from 'react';

import {SuccessIcon, WarningIcon} from '../../shared/components/fa-icons';

export function ScheduleValidator({schedule}: {schedule: string}) {
    try {
        if (schedule.split(' ').length >= 6) {
            throw new Error('cron schedules must consist of 5 values only');
        }
        return (
            <span>
                <SuccessIcon /> {x.toString(schedule)}
            </span>
        );
    } catch (e) {
        return (
            <span>
                <WarningIcon /> Schedule maybe invalid: {e.toString()}
            </span>
        );
    }
}
