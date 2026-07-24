import {matchesNameFilter} from './workflow-filters';

describe('matchesNameFilter', () => {
    describe('Contains filter', () => {
        test('returns true when name contains the value', () => {
            expect(matchesNameFilter('my-workflow-test', 'workflow', 'Contains')).toBe(true);
        });

        test('returns false when name does not contain the value', () => {
            expect(matchesNameFilter('my-workflow-test', 'other', 'Contains')).toBe(false);
        });
    });

    describe('Prefix filter', () => {
        test('returns true when name starts with the value', () => {
            expect(matchesNameFilter('my-workflow-test', 'my-', 'Prefix')).toBe(true);
        });

        test('returns false when name does not start with the value', () => {
            expect(matchesNameFilter('my-workflow-test', 'workflow', 'Prefix')).toBe(false);
        });
    });

    describe('Exact filter', () => {
        test('returns true when name equals the value', () => {
            expect(matchesNameFilter('my-workflow', 'my-workflow', 'Exact')).toBe(true);
        });

        test('returns false when name does not equal the value', () => {
            expect(matchesNameFilter('my-workflow-test', 'my-workflow', 'Exact')).toBe(false);
        });
    });

    describe('NotEquals filter', () => {
        test('returns true when name does not equal the value', () => {
            expect(matchesNameFilter('my-workflow-test', 'other-workflow', 'NotEquals')).toBe(true);
        });

        test('returns false when name equals the value', () => {
            expect(matchesNameFilter('my-workflow', 'my-workflow', 'NotEquals')).toBe(false);
        });
    });

    describe('empty nameValue', () => {
        test('returns true for any name when nameValue is empty', () => {
            expect(matchesNameFilter('any-workflow', '', 'Contains')).toBe(true);
            expect(matchesNameFilter('any-workflow', '', 'Prefix')).toBe(true);
            expect(matchesNameFilter('any-workflow', '', 'Exact')).toBe(true);
            expect(matchesNameFilter('any-workflow', '', 'NotEquals')).toBe(true);
        });
    });
});
