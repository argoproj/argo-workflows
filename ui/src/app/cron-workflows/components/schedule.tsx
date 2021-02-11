import React = require('react');
import {SuccessIcon, WarningIcon} from '../../shared/components/fa-icons';

const x = require('cronstrue');

/*
    https://github.com/bradymholt/cRonstrue
    vs
    https://github.com/robfig/cron

    I think we must assume that these libraries (or any two libraries) will never be exactly the same and accept that
    sometime it'll not work as expected. Therefore, we must let the user know about this.
 */

export const Schedule = ({schedule}: {schedule: string}) => {
    try {
        const pretty = x.toString(schedule);
        return (
            <span title={pretty}>
                <code>{schedule}</code> {pretty}
            </span>
        );
    } catch (e) {
        return <>schedule</>;
    }
};

export const ScheduleValidator = ({schedule}: {schedule: string}) => {
    try {
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
};
