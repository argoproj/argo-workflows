import {uiUrl} from './base';

export const getCookie = (name: string) =>
    (
        decodeURIComponent(document.cookie)
            .split(';')
            .map(x => x.trim())
            .find(x => x.startsWith(name + '=')) || ''
    ).replace(/^.*="?(.*?)"?$/, '$1');

export function setCookie(name: string, value: string) {
    document.cookie = name + '=' + value + ';SameSite=Strict;path=' + uiUrl('');
}

export function deleteCookie(name: string) {
    // "If the user agent receives a new cookie with the same cookie-name,
    // domain-value, and path-value as a cookie that it has already stored, the
    // existing cookie is evicted and replaced with the new cookie. Notice that
    // servers can delete cookies by sending the user agent a new cookie with an
    // Expires attribute with a value in the past."
    // Spec: https://httpwg.org/specs/rfc6265.html#sane-set-cookie-semantics
    document.cookie = name + '=;Max-Age=0;path=' + uiUrl('');
}
