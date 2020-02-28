// requested resource is seconds
export function formatDuration(d: number) {
    if (d > 86400) {
        return d / 86400 + 'd';
    }
    if (d > 3600) {
        return d / 3600 + 'h';
    }
    if (d > 60) {
        return d / 60 + 'm';
    }
    return d + 's';
}
