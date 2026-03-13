import {uiUrl} from './base';
import * as nsUtils from './namespaces';

/**
 * Return a URL suitable to use with `history.push(..)`. Optionally saving the "namespace" parameter as the current namespace.
 * Only "truthy" values are put into the query parameters. I.e. "falsey" values include null, undefined, false, "", 0.
 */
export function historyUrl(path: string, params: {[key: string]: any}) {
    const queryParams = new URLSearchParams();
    Object.entries(params)
        .filter(([, v]) => v !== null)
        .forEach(([k, v]) => {
            const searchValue = '{' + k + '}';
            if (path.includes(searchValue)) {
                path = path.replace(searchValue, v != null ? v : '');
            } else if (k === 'extraSearchParams') {
                (v as URLSearchParams).forEach((value, key) => queryParams.append(key, value));
            } else if (v) {
                queryParams.set(k, v);
            }
            if (k === 'namespace') {
                nsUtils.setCurrentNamespace(v);
            }
        });

    return uiUrl(path.replace(/{[^}]*}/g, '')) + '?' + queryParams.toString();
}
