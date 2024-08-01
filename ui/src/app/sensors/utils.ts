import {Condition} from '../../models';

export function statusIconClasses(conditions: Condition[], icon: string): string {
    let classes = [icon];
    if (!conditions || conditions.length === 0) {
        classes = [icon, 'status-icon--init'];
        return classes.join(' ');
    }

    for (const condition of conditions) {
        if (condition.status === 'False') {
            classes = [icon, 'status-icon--failed'];
            return classes.join(' ');
        } else if (condition.status === 'Unknown') {
            classes = [icon, 'status-icon--spin'];
            return classes.join(' ');
        }
    }

    classes = [icon, 'status-icon--running'];
    return classes.join(' ');
}
