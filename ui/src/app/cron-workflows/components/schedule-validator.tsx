import React = require('react');
import {SuccessIcon, WarningIcon} from '../../shared/components/fa-icons';

const x = require('cronstrue');

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
