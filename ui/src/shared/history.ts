import {uiUrl} from './base';
import * as nsUtils from './namespaces';

/**
 * Return a URL suitable to use with `history.push(..)`. Optionally saving the "namespace" parameter as the current namespace.
 * Only "truthy" values are put into the query parameters. I.e. "falsey" values include null, undefined, false, "", 0.
 */
export function historyUrl(path: string, params: {[key: string]: any}) {
    const queryParams = new URLSearchParams();
    let extraSearchParams: URLSearchParams | undefined;

    // Process named params first so that the namespace dedup check below works
    // regardless of key order in the params object.
    Object.entries(params)
        .filter(([, v]) => v !== null)
        .forEach(([k, v]) => {
            if (k === 'extraSearchParams') {
                extraSearchParams = v as URLSearchParams;
                return;
            }
            const searchValue = '{' + k + '}';
            if (path.includes(searchValue)) {
                path = path.replace(searchValue, v != null ? v : '');
            } else if (v) {
                queryParams.set(k, v);
            }
            if (k === 'namespace') {
                nsUtils.setCurrentNamespace(v);
            }
        });

    // Append extraSearchParams after named params. Skip namespace if it was
    // already set as a named query param to prevent duplicate ?namespace= entries.
    // Repeated values for the same key within extraSearchParams are preserved
    // (e.g. multiple phase= or label=).
    extraSearchParams?.forEach((value, key) => {
        if (key !== 'namespace' || !queryParams.has('namespace')) {
            queryParams.append(key, value);
        }
    });

    return uiUrl(path.replace(/{[^}]*}/g, '')) + '?' + queryParams.toString();
}
