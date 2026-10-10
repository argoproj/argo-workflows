import x from 'cronstrue';
import * as React from 'react';

import {SuccessIcon, WarningIcon} from '../shared/components/fa-icons';
import {isHashedSchedule, validateHashedSchedule} from '../shared/cron';

export function ScheduleValidator({schedule}: {schedule: string}) {
    try {
        if (schedule.split(' ').length >= 6) {
            throw new Error('cron schedules must consist of 5 values only');
        }
        if (isHashedSchedule(schedule)) {
            const error = validateHashedSchedule(schedule);
            if (error) {
                throw new Error(error);
            }
            return (
                <span>
                    <SuccessIcon /> Hashed schedule, resolved from the name and namespace of the CronWorkflow
                </span>
            );
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
