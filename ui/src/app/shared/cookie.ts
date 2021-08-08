import {uiUrl} from './base';

export const authCookieName = 'authorization';
export const serviceAccountHintCookieName = 'service-account-hint';

export function setCookie(name: string, value: string) {
    const path = uiUrl('');
    document.cookie = `${name}=${value};SameSite=Strict;path=${path}`;
}

export function clearCookie(name: string) {
    document.cookie = `${name}=;Max-Age=0`;
}
