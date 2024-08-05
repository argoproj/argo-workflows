export const AUTH_COOKIE = 'authorization';

export function getCookie(name: string) {
    return (
        decodeURIComponent(document.cookie)
            .split(';')
            .map(x => x.trim())
            .find(x => x.startsWith(name + '=')) || ''
    ).replace(/^.*="?(.*?)"?$/, '$1');
}

export function setCookie(name: string, value: string, path: string) {
    document.cookie = name + '=' + value + ';SameSite=Strict;path=' + path;
}

export function resetCookie(name: string) {
    document.cookie = `${name}=;Max-Age=0`;
}
