import {ProcessURL} from './links';

describe('process URL', () => {
    test('no filter', () => {
        const object = {
            status: {
                startedAt: '2021-01-01T10:30:00Z',
                finishedAt: '2021-01-01T10:30:00Z'
            }
        };
        expect(ProcessURL('https://logging?from=${status.startedAt}&to=${status.finishedAt}', object)).toBe('https://logging?from=2021-01-01T10:30:00Z&to=2021-01-01T10:30:00Z');
    });

    test('with filter', () => {
        const object = {
            status: {
                startedAt: '2021-01-01T10:30:00Z',
                finishedAt: '2021-01-01T10:30:00Z'
            }
        };
        expect(ProcessURL('https://logging?from=${status.startedAt | date: "%s"}&to=${status.finishedAt | date: "%s"}', object)).toBe(
            'https://logging?from=1609497000&to=1609497000'
        );
    });

    test('ongoing workflow with filter', () => {
        const object = {
            status: {
                startedAt: '2021-01-01T10:30:00Z',
                finishedAt: ''
            }
        };
        expect(ProcessURL('https://logging?from=${status.startedAt | date: "%s"}&to=${status.finishedAt | date: "%s" | default: "now"}', object)).toBe(
            'https://logging?from=1609497000&to=now'
        );
    });
});
