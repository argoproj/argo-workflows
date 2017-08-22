import { Injectable } from '@angular/core';

/**
 * Service to enable cookie management for the application
 *
 * Default cookie is just a session cookie
 */
@Injectable()
export class CookieService {
    writeCookie(name: string, value: string, domain?: string, days?: number) {
        let path = '; path=/',
            // This will come into picture with .magnifi.fm domains only.
            // In local env the cookies will not be set
            defaultCookieDomain = window.location.hostname || '',
            expires = '',
            https = window.location.protocol === 'https:',
            secure,
            isLocalServer = window.location.hostname.indexOf('localhost') > -1,
            domainPart = '',
            cookieString = '';

        if (!!days) {
            let date = new Date();
            date.setTime(date.getTime() + (days * 24 * 60 * 60 * 1000));
            expires = '; expires=' + date.toUTCString();
        }

        secure = https ? ';secure' : '';

        domainPart = (!domain) ? '; domain=' + defaultCookieDomain : '; domain=' + domain;
        // fixes IE edge issue in localhost environment.
        domainPart = isLocalServer ? '' : domainPart;

        cookieString = name + '=' + encodeURIComponent(value) + domainPart + expires + secure + path;
        document.cookie = cookieString;
    }

    readCookie(name: string) {
        let value = document.cookie.match(new RegExp(name + '=([^;]+)')) || [];
        return decodeURIComponent(value[1] || '');
    }

    deleteCookie(name: string, domain?: string) {
        // domain is compatible similar to writeCookie
        let d = (!domain) ? '' : domain;
        this.writeCookie(name, null, d, -1);
    }

    getSessionToken() {
        return this.readCookie('session_token');
    }

    deleteSessionToken() {
        this.deleteCookie('session_token');
    }

    clearSessionCookies() {
        this.deleteSessionToken();
    }
}
