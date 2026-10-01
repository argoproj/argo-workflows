import parser from 'cron-parser';

import {CronWorkflow} from './models';

// the ranges of cron itself, which an explicit `H(<min>-<max>)` has to stay within
const FIELD_LIMITS = [
    [0, 59], // minute
    [0, 23], // hour
    [1, 31], // day of month
    [1, 12], // month
    [0, 6] // day of week
];
const FIELD_NAMES = ['minute', 'hour', 'day of month', 'month', 'day of week'];
const HASH_SYNTAX = '`H`, `H/<step>`, `H(<min>-<max>)` or `H(<min>-<max>)/<step>`';

export function getNextScheduledTime(schedule: string, tz: string): Date {
    let out: Date;
    try {
        out = parser.parseExpression(schedule, {utc: !tz, tz}).next().toDate();
    } catch (e) {
        // Do nothing
    }
    return out;
}

// A hashed schedule uses `H` in at least one field, e.g. `H H * * *`. The controller resolves it
// from the name and namespace of the CronWorkflow, so it cannot be parsed here.
export function isHashedSchedule(schedule: string): boolean {
    return schedule.split(/\s+/).some(field => field.split(',').some(item => item.startsWith('H')));
}

// getResolvedSchedules returns the schedules the CronWorkflow runs on, i.e. with every `H` resolved.
// The status is keyed by the configured schedule, so an edited schedule falls back to itself rather
// than to the resolution of the schedule it replaced.
export function getResolvedSchedules(wf: CronWorkflow): string[] {
    const resolved = wf.status?.resolvedSchedules;
    return (wf.spec.schedules ?? []).map(schedule => resolved?.[schedule] ?? schedule);
}

// validateHashedSchedule mirrors the `H` syntax the controller accepts, so that the editor does not
// accept a schedule the API rejects. It returns an error message, or null if the schedule is fine.
export function validateHashedSchedule(schedule: string): string | null {
    const fields = schedule.trim().split(/\s+/);
    // anything which is not a five field expression is left to the cron parser
    if (fields.length !== FIELD_LIMITS.length) {
        return null;
    }
    for (let i = 0; i < fields.length; i++) {
        for (const item of fields[i].split(',')) {
            const error = item.startsWith('H') && validateHashedItem(item, i);
            if (error) {
                return error;
            }
        }
    }
    return null;
}

function validateHashedItem(item: string, fieldIndex: number): string | null {
    let rest = item.slice(1);

    if (rest.startsWith('(')) {
        const end = rest.indexOf(')');
        if (end < 0) {
            return `"${item}" is malformed: missing \`)\``;
        }
        const error = validateRange(rest.slice(1, end), fieldIndex);
        if (error) {
            return `"${item}" is malformed: ${error}`;
        }
        rest = rest.slice(end + 1);
    }

    if (rest === '') {
        return null;
    }
    const step = rest.startsWith('/') ? parseIntStrict(rest.slice(1)) : null;
    if (step === null) {
        return `"${item}" is malformed: expected ${HASH_SYNTAX}`;
    }
    if (step <= 0) {
        return `"${item}" is malformed: step must be greater than 0`;
    }
    return null;
}

function validateRange(range: string, fieldIndex: number): string | null {
    const separator = range.indexOf('-');
    const min = separator < 0 ? null : parseIntStrict(range.slice(0, separator));
    const max = separator < 0 ? null : parseIntStrict(range.slice(separator + 1));
    if (min === null || max === null) {
        return `expected a range like \`1-5\`, got "${range}"`;
    }
    if (min > max) {
        return `range "${range}" is inverted`;
    }
    const [limitMin, limitMax] = FIELD_LIMITS[fieldIndex];
    if (min < limitMin || max > limitMax) {
        return `range "${range}" is outside of ${limitMin}-${limitMax}, the range of the ${FIELD_NAMES[fieldIndex]} field`;
    }
    return null;
}

function parseIntStrict(value: string): number | null {
    return /^-?\d+$/.test(value) ? Number(value) : null;
}
