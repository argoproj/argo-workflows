import {uiUrl} from './base';
import {Utils} from './utils';

export const toHistory = (path: string, params: {[key: string]: any}) => {
    const queryParams: string[] = [];
    Object.entries(params)
        .filter(([, v]) => v !== null)
        .forEach(([k, v]) => {
            const searchValue = '{' + k + '}';
            if (path.includes(searchValue)) {
                path = path.replace(searchValue, v);
            } else if (v) {
                queryParams.push(k + '=' + v);
            }
            if (k === 'namespace') {
                Utils.setCurrentNamespace(v);
            }
        });
    return uiUrl(path) + '?' + queryParams.join('&');
};
