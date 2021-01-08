import {Sensor} from '../../../models';

export const Utils = {
    statusIconClasses(sensor: Sensor): string {
        let classes = ['fa-circle'];
        if (!sensor.status || !sensor.status.conditions || sensor.status.conditions.length === 0) {
            classes = ['fa-circle', 'status-icon--init'];
        } else {
            let isRunning = false;
            let hasFailed = false;
            sensor.status.conditions.map(condition => {
                if (condition.status === 'False') {
                    hasFailed = true;
                } else if (condition.status === 'Unknown') {
                    isRunning = true;
                }
            });
            if (hasFailed) {
                classes = ['fa-circle', 'status-icon--failed'];
            } else if (isRunning) {
                classes = ['fa-circle', 'status-icon--spin'];
            } else {
                classes = ['fa-circle', 'status-icon--success'];
            }
        }
        return classes.join(' ');
    }
};
