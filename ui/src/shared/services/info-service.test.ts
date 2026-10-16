/**
 * @jest-environment jsdom
 */
import {Link} from '../models';
import {resolveLinks} from './info-service';

function setBase(href: string) {
    document.head.innerHTML = `<base href="${href}">`;
}

const link = (url: string): Link => ({name: 'test', scope: 'workflow-list', url});

describe('resolveLinks', () => {
    test('absolute links are left unchanged', () => {
        setBase('/');
        expect(resolveLinks([link('https://example.com/foo?bar=baz')])[0].url).toBe('https://example.com/foo?bar=baz');
    });

    test('relative link with a leading slash on the default base', () => {
        setBase('/');
        // must not become "//my-namespace", which the browser treats as scheme-relative
        expect(resolveLinks([link('/my-namespace')])[0].url).toBe('/my-namespace');
    });

    test('relative link with a leading slash on a mounted base', () => {
        setBase('/argo/');
        expect(resolveLinks([link('/my-namespace')])[0].url).toBe('/argo/my-namespace');
    });

    test('query-only relative link is prefixed with the base', () => {
        setBase('/argo/');
        expect(resolveLinks([link('?namespace=argo&phase=Failed')])[0].url).toBe('/argo/?namespace=argo&phase=Failed');
    });

    test('undefined links yields an empty array', () => {
        setBase('/');
        expect(resolveLinks(undefined)).toEqual([]);
    });
});
