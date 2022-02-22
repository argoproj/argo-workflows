import React = require('react');
import {WarningIcon} from '../../shared/components/fa-icons';

const x = require('cronstrue');

/*
    https://github.com/bradymholt/cRonstrue
    vs
    https://github.com/robfig/cron

    I think we must assume that these libraries (or any two libraries) will never be exactly the same and accept that
    sometime it'll not work as expected. Therefore, we must let the user know about this.
 */

export const PrettySchedule = ({schedule}: {schedule: string}) => {
    try {
        if (schedule.split(' ').length >= 6) {
            throw new Error('cron schedules must consist of 5 values only');
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
};
