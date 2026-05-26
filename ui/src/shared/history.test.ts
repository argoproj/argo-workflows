/**
 * @jest-environment jsdom
 */
import {historyUrl} from './history';
import * as nsUtils from './namespaces';

describe('history URL', () => {
    test('namespace', () => {
        expect(historyUrl('foo/{namespace}', {namespace: 'my-ns'})).toBe('/foo/my-ns?');
        expect(nsUtils.getCurrentNamespace()).toBe('my-ns');
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

    test('repeated extra search parameters', () => {
        const params = new URLSearchParams();
        params.append('label', 'a');
        params.append('label', 'b');
        expect(historyUrl('foo', {extraSearchParams: params})).toBe('/foo?label=a&label=b');
    });

    test('namespace in extraSearchParams is ignored when already set via named param', () => {
        const params = new URLSearchParams();
        params.append('namespace', 'stale');
        params.append('label', 'a');
        // namespace named param takes precedence; 'stale' from extraSearchParams is dropped
        expect(historyUrl('foo', {namespace: 'argo', extraSearchParams: params})).toBe('/foo?namespace=argo&label=a');
    });
});
