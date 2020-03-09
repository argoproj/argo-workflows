/**
 * Format the given number number of seconds in the form _d_h_m_s.
 * @param seconds Number of seconds to format. Will be rounded to the nearest whole number.
 */
export function formatDuration(seconds: number) {
    let remainingSeconds = Math.round(seconds);
    let formattedDuration = '';

    if (remainingSeconds > 86400) {
        formattedDuration += Math.floor(remainingSeconds / 86400) + 'd';
        remainingSeconds = remainingSeconds % 86400;
    }
    if (remainingSeconds > 3600) {
        formattedDuration += Math.floor(remainingSeconds / 3600) + 'h';
        remainingSeconds = remainingSeconds % 3600;
    }
    if (remainingSeconds > 60) {
        formattedDuration += Math.floor(remainingSeconds / 60) + 'm';
        remainingSeconds = remainingSeconds % 60;
    }
    if (remainingSeconds > 0 || Math.round(seconds) === 0) {
        formattedDuration += remainingSeconds + 's';
    }

    return formattedDuration;
}
