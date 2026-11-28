import {getResolvedSchedules, isHashedSchedule, validateHashedSchedule} from './cron';
import {CronWorkflow} from './models';

function cronWorkflow(schedules: string[], resolvedSchedules?: {[schedule: string]: string}): CronWorkflow {
    return {
        metadata: {
            name: 'my-cron-workflow',
            namespace: 'default'
        },
        spec: {schedules, workflowSpec: {}},
        status: resolvedSchedules ? {active: [], lastScheduledTime: null, resolvedSchedules} : undefined
    };
}

describe('isHashedSchedule', () => {
    test.each(['H * * * *', '* H * * *', 'H/15 * * * *', 'H(0-29)/10 * * * *', '0,H * * * *'])('%s is hashed', schedule => {
        expect(isHashedSchedule(schedule)).toBe(true);
    });

    test.each(['0 * * * *', '* * * * *', '@daily', '0 0 * * MON-FRI', '0 0 * * THU'])('%s is not hashed', schedule => {
        expect(isHashedSchedule(schedule)).toBe(false);
    });
});

describe('validateHashedSchedule', () => {
    test.each(['H * * * *', 'H,30 H(9-17) * * MON-FRI', 'H/15 * * * *', 'H(0-29)/10 * * * *', '0 0 H(29-31) * *', '0 0 * * H(0-6)'])('%s is accepted', schedule => {
        expect(validateHashedSchedule(schedule)).toBeNull();
    });

    // the messages are the ones the controller reports, so the editor and the API agree
    test.each([
        ['H(0-30 * * * *', '"H(0-30" is malformed: missing `)`'],
        ['H(30-10) * * * *', '"H(30-10)" is malformed: range "30-10" is inverted'],
        ['H(0-90) * * * *', '"H(0-90)" is malformed: range "0-90" is outside of 0-59, the range of the minute field'],
        ['* H(0-30) * * *', '"H(0-30)" is malformed: range "0-30" is outside of 0-23, the range of the hour field'],
        ['* * H(0-31) * *', '"H(0-31)" is malformed: range "0-31" is outside of 1-31, the range of the day of month field'],
        ['* * * * H(0-7)', '"H(0-7)" is malformed: range "0-7" is outside of 0-6, the range of the day of week field'],
        ['H(a-b) * * * *', '"H(a-b)" is malformed: expected a range like `1-5`, got "a-b"'],
        ['H/0 * * * *', '"H/0" is malformed: step must be greater than 0'],
        ['H15 * * * *', '"H15" is malformed: expected `H`, `H/<step>`, `H(<min>-<max>)` or `H(<min>-<max>)/<step>`']
    ])('%s is rejected', (schedule, expected) => {
        expect(validateHashedSchedule(schedule)).toBe(expected);
    });

    it('leaves expressions which are not five fields to the cron parser', () => {
        expect(validateHashedSchedule('H H * *')).toBeNull();
    });
});

describe('getResolvedSchedules', () => {
    it('uses the resolved schedules of the status', () => {
        expect(getResolvedSchedules(cronWorkflow(['H H * * *', '0 * * * *'], {'H H * * *': '9 6 * * *'}))).toEqual(['9 6 * * *', '0 * * * *']);
    });

    it('keeps a schedule which contains a comma itself separate', () => {
        expect(getResolvedSchedules(cronWorkflow(['H,30 * * * *'], {'H,30 * * * *': '9,30 * * * *'}))).toEqual(['9,30 * * * *']);
    });

    it('falls back to the configured schedules while the status is missing', () => {
        expect(getResolvedSchedules(cronWorkflow(['H H * * *']))).toEqual(['H H * * *']);
    });

    it('does not use the resolution of a schedule which has since been edited', () => {
        expect(getResolvedSchedules(cronWorkflow(['0 0 * * *'], {'H H * * *': '9 6 * * *'}))).toEqual(['0 0 * * *']);
    });
});
