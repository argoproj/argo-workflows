// usage is seconds
export function formatUsageIndicator(u: number) {
    if (u > 86400) {
        return u / 86400 + 'd';
    }
    if (u > 3600) {
        return u / 3600 + 'h';
    }
    if (u > 60) {
        return u / 60 + 'm';
    }
    return u + 's';
}
