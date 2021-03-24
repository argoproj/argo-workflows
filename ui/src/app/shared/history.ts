import {uiUrl} from './base';
import {Utils} from './utils';

/**
 * Return a URL suitable to use with `history.push(..)`. Optionally saving the "namespace" parameter as the current namespace.
 * Only "truthy" values are put into the query parameters. I.e. "falsey" values include null, undefined, false, "", 0.
 */
export const historyUrl = (path: string, params: {[key: string]: any}) => {
    const queryParams: string[] = [];
    Object.entries(params)
        .filter(([, v]) => v !== null)
        .forEach(([k, v]) => {
            const searchValue = '{' + k + '}';
            if (path.includes(searchValue)) {
                path = path.replace(searchValue, v != null ? v : '');
            } else if (v) {
                queryParams.push(k + '=' + v);
            }
            if (k === 'namespace') {
                Utils.currentNamespace = v;
            }
        });
    return uiUrl(path.replace(/{[^}]*}/g, '')) + '?' + queryParams.join('&');
};
