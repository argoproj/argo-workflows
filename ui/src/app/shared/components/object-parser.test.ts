import {isEqual} from './object-parser';

describe('isEqual', () => {
    test('two objects', () => {
        expect(isEqual({}, {})).toBe(true);
        expect(isEqual({a: 1, b: 2}, {a: 1, b: 2})).toBe(true);
        expect(isEqual({a: 1, b: 2}, {a: 1, b: 3})).toBe(false);
        expect(isEqual({a: 1, b: 2}, {a: 1, c: 2})).toBe(false);
    });

    test('two strings', () => {
        expect(isEqual('foo', 'foo')).toBe(true);
        expect(isEqual('foo', 'bar')).toBe(false);
        expect(isEqual('', 'bar')).toBe(false);
        expect(isEqual('', '')).toBe(true);
    });
});
