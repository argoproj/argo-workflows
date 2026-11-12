import x from 'cronstrue';
import * as React from 'react';

import {WarningIcon} from '../shared/components/fa-icons';
import {isHashedSchedule, validateHashedSchedule} from '../shared/cron';

/*
    https://github.com/bradymholt/cRonstrue
    vs
    https://github.com/robfig/cron

    I think we must assume that these libraries (or any two libraries) will never be exactly the same and accept that
    sometime it'll not work as expected. Therefore, we must let the user know about this.
 */

export function PrettySchedule({schedule}: {schedule: string}) {
    try {
        if (schedule.split(' ').length >= 6) {
            throw new Error('cron schedules must consist of 5 values only');
        } else if (schedule.startsWith('@every')) {
            return null;
        } else if (isHashedSchedule(schedule)) {
            const error = validateHashedSchedule(schedule);
            if (error) {
                throw new Error(error);
            }
            // the controller resolves it, so it is only shown once the CronWorkflow has been scheduled
            return <span title='Resolved from the name and namespace of the CronWorkflow'>hashed schedule</span>;
        }

        const pretty = x.toString(schedule);
        return <span title={pretty}>{pretty}</span>;
    } catch (e) {
        return (
            <span>
                <WarningIcon /> {e.toString()}
            </span>
        );
    }
}
