import * as models from '../../models';

/**
 * Format the given number number of seconds in the form _d_h_m_s.
 * @param seconds Number of seconds to format. Will be rounded to the nearest whole number.
 * @param sigfigs Level of significant figures to show.
 *  Here, a sigfig is one of days, hours, minutes, or seconds.
 *  Examples: 13h has one sigfig and 13h22s has two sigfigs.
 */

export function formatDuration(seconds: number, sigfigs = 2) {
    let remainingSeconds = Math.abs(Math.round(seconds));
    let formattedDuration = '';
    const figs = [];

    if (remainingSeconds > 86400) {
        const days = Math.floor(remainingSeconds / 86400) + 'd';
        figs.push(days);
        formattedDuration += days;
        remainingSeconds = remainingSeconds % 86400;
    }

    if (remainingSeconds > 3600) {
        const hours = Math.floor(remainingSeconds / 3600) + 'h';
        figs.push(hours);
        formattedDuration += hours;
        remainingSeconds = remainingSeconds % 3600;
    }

    if (remainingSeconds > 60) {
        const minutes = Math.floor(remainingSeconds / 60) + 'm';
        figs.push(minutes);
        formattedDuration += minutes;
        remainingSeconds = remainingSeconds % 60;
    }

    if (remainingSeconds > 0 || Math.round(seconds) === 0) {
        figs.push(remainingSeconds + 's');
        formattedDuration += remainingSeconds + 's';
    }

    if (sigfigs <= figs.length) {
        formattedDuration = '';
        for (let i = 0; i < sigfigs; i++) {
            formattedDuration += figs[i];
        }
        return formattedDuration;
    }

    return formattedDuration;
}

export function denominator(resource: string) {
    switch (resource) {
        case 'memory':
            return '100Mi';
        case 'storage':
            return '10Gi';
        case 'ephemeral-storage':
            return '10Gi';
        default:
            return '1';
    }
}

export function wfDuration(status: models.WorkflowStatus) {
    if (!status.startedAt) {
        return 0;
    }
    return ((status.finishedAt ? new Date(status.finishedAt) : new Date()).getTime() - new Date(status.startedAt).getTime()) / 1000;
}

export const ago = (date: Date) => {
    const secondsAgo = (new Date().getTime() - date.getTime()) / 1000;
    const duration = formatDuration(secondsAgo);
    if (secondsAgo < 0) {
        return 'in ' + duration;
    } else {
        return duration + ' ago';
    }
};
