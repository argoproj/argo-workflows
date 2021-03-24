import {Condition} from '../../../models';

export const Utils = {
    statusIconClasses(conditions: Condition[], icon: string): string {
        let classes = [icon];
        if (!conditions || conditions.length === 0) {
            classes = [icon, 'status-icon--init'];
        } else {
            let isRunning = false;
            let hasFailed = false;
            conditions.map(condition => {
                if (condition.status === 'False') {
                    hasFailed = true;
                } else if (condition.status === 'Unknown') {
                    isRunning = true;
                }
            });
            if (hasFailed) {
                classes = [icon, 'status-icon--failed'];
            } else if (isRunning) {
                classes = [icon, 'status-icon--spin'];
            } else {
                classes = [icon, 'status-icon--running'];
            }
        }
        return classes.join(' ');
    }
};
