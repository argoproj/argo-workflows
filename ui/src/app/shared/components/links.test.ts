import {ProcessURL} from './links';

describe('process URL', () => {
    test('original timestamp', () => {
        const object = {
            status: {
                startedAt: '2021-01-01T10:30:00Z',
                finishedAt: '2021-01-01T10:30:00Z'
            }
        };
        expect(ProcessURL('https://logging?from=$%7Bstatus.startedAt%7D&to=$%7Bstatus.finishedAt%7D', object)).toBe(
            'https://logging?from=2021-01-01T10:30:00Z&to=2021-01-01T10:30:00Z'
        );
    });

    test('url encoded string', () => {
        const object = {
            metadata: {
                name: 'test'
            },
            status: {
                startedAt: '2021-01-01T10:30:00Z'
            }
        };
        expect(ProcessURL('https://logging/$%7Bmetadata.name%7D', object)).toBe('https://logging/test');
    });

    test('epoch timestamp', () => {
        const object = {
            status: {
                startedAt: '2021-01-01T10:30:00Z',
                finishedAt: '2021-01-01T10:30:00Z'
            }
        };
        expect(ProcessURL('https://logging?from=$%7Bstatus.startedAtEpoch%7D&to=$%7Bstatus.finishedAtEpoch%7D', object)).toBe(
            'https://logging?from=1609497000000&to=1609497000000'
        );
    });

    test('epoch timestamp with ongoing workflow', () => {
        const object = {
            status: {
                startedAt: '2021-01-01T10:30:00Z'
            }
        };

        const expectedDate = new Date('2021-03-01T10:30:00.00Z');
        jest.spyOn(global.Date, 'now').mockImplementationOnce(() => expectedDate.valueOf());

        expect(ProcessURL('https://logging?from=$%7Bstatus.startedAtEpoch%7D&to=$%7Bstatus.finishedAtEpoch%7D', object)).toBe(
            `https://logging?from=1609497000000&to=${expectedDate.getTime()}`
        );
    });

    test('no timestamp', () => {
        const object = {
            status: {}
        };

        expect(ProcessURL('https://logging?from=$%7Bstatus.startedAtEpoch%7D&to=$%7Bstatus.finishedAtEpoch%7D', object)).toBe(`https://logging?from=null&to=null`);
    });
});
