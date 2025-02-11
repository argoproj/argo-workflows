import parser from 'cron-parser';

export function getNextScheduledTime(schedule: string, tz: string): Date {
    let out: Date;
    try {
        out = parser
            .parseExpression(schedule, {utc: !tz, tz})
            .next()
            .toDate();
    } catch (e) {
        // Do nothing
    }
    return out;
}
