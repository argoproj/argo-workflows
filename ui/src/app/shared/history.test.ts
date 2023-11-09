/**
 * @jest-environment jsdom
 */
import {historyUrl} from './history';
import {Utils} from './utils';

describe('history URL', () => {
    test('namespace', () => {
        expect(historyUrl('foo/{namespace}', {namespace: 'my-ns'})).toBe('/foo/my-ns?');
        expect(Utils.currentNamespace).toBe('my-ns');
    });

    test('path parameter', () => {
        expect(historyUrl('foo/{bar}', {bar: 'baz'})).toBe('/foo/baz?');
    });

    test('null/undefined path parameter', () => {
        expect(historyUrl('foo/{bar}', {bar: null})).toBe('/foo/?');
        expect(historyUrl('foo/{bar}', {})).toBe('/foo/?');
        expect(historyUrl('foo/{bar}/{baz}', {})).toBe('/foo//?');
    });

    test('query parameter', () => {
        expect(historyUrl('foo', {bar: 'baz'})).toBe('/foo?bar=baz');
    });

    test('falsey query parameter', () => {
        expect(historyUrl('foo', {bar: false})).toBe('/foo?');
        expect(historyUrl('foo', {bar: null})).toBe('/foo?');
    });
});
