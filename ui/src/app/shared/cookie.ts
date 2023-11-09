import {uiUrl} from './base';

export const getCookie = (name: string) =>
    (
        decodeURIComponent(document.cookie)
            .split(';')
            .map(x => x.trim())
            .find(x => x.startsWith(name + '=')) || ''
    ).replace(/^.*="?(.*?)"?$/, '$1');

export const setCookie = (name: string, value: string) => {
    document.cookie = name + '=' + value + ';SameSite=Strict;path=' + uiUrl('');
};
