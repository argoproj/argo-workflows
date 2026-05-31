import {CronExpressionParser} from 'cron-parser';

export function getNextScheduledTime(schedule: string, tz: string): Date {
    let out: Date;
    try {
        out = CronExpressionParser.parse(schedule, {tz: tz || 'UTC'})
            .next()
            .toDate();
    } catch {
        // Do nothing
    }
    return out;
}
