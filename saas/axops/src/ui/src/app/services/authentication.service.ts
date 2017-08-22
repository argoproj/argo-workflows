import { Injectable, EventEmitter } from '@angular/core';
import { LocationStrategy } from '@angular/common';
import { Http } from '@angular/http';
import { Observable } from 'rxjs/Rx';
import { Router } from '@angular/router';
import { CookieService } from './cookie.service';
import { UsersService } from './users.service';
import { TrackingService } from './tracking.service';
import { AxHeaders } from './headers';
import { User, USER_GROUPS } from '../model';

const CURRENT_USER_KEY = 'ax-current-user';

@Injectable()
export class AuthenticationService {
    public onUserUpdated: EventEmitter<User> = new EventEmitter<User>();

    private currentUserPromise: Promise<User> = null;

    private get currentUser(): User {
        let userData: any;
        let userDataJson = localStorage.getItem(CURRENT_USER_KEY);
        if (userDataJson) {
            try {
                userData = JSON.parse(userDataJson);
            } catch (_) {
                userData = null;
            }
        }
        return userData ? new User(userData) : null;
    }

    private set currentUser(val: User) {
        if (!val) {
            localStorage.removeItem(CURRENT_USER_KEY);
        } else {
            localStorage.setItem(CURRENT_USER_KEY, JSON.stringify(val));
            this.trackingService.initialize(val);
        }
    }

    constructor(
            private http: Http,
            private router: Router,
            private cookieService: CookieService,
            private usersService: UsersService,
            private trackingService: TrackingService,
            private locationStrategy: LocationStrategy) {
        this.currentUser = null;

        $(window).bind('storage', e => {
             if (e.originalEvent['key'] === CURRENT_USER_KEY) {
                 let user = this.currentUser;
                 if (user != null) {
                     this.onUserUpdated.emit(user);
                 }
             }
        });
    }

    public getCurrentUser(): Promise<User> {
        if (this.currentUser === null) {
            if (!this.currentUserPromise) {
                this.currentUserPromise = new Promise(async (resolve, reject) => {
                    try {
                        let res = await this.http.get('v1/users/session', {headers: new AxHeaders({ noErrorHandling: true })} ).toPromise();
                        this.currentUser = new User(res.json());
                        resolve(this.currentUser);
                    } catch (e) {
                        let isAnonymousUserSupported = await this.isAnonymousUserSupported();
                        if (isAnonymousUserSupported) {
                            this.currentUser = new User({
                                first_name: 'Anonymous',
                                last_name: 'Anonymous',
                                anonymous: true,
                                view_preferences: { isIntroductionCompleted: 'true' },
                                groups: [ USER_GROUPS.developer ],
                            });
                            resolve(this.currentUser);
                        } else {
                            reject(e);
                        }
                    }
                });
            }
            return this.currentUserPromise;
        }
        return Promise.resolve(this.currentUser);
    }

    public updateCurrentUser(callback: (user: User) => void, hideLoader?: boolean): Promise<User> {
        return this.getCurrentUser().then(user => {
            callback(user);
            this.currentUser = user;
            if (user.anonymous) {
                return Promise.resolve(user);
            } else {
                return this.usersService.updateUser(user, hideLoader).toPromise().then(() => user);
            }
        });
    }

    public login(username, password) {
        this.clearUserSession();
        return this.http.post('v1/auth/login', JSON.stringify({
            username: username,
            password: password
        }), { headers: new AxHeaders({ noErrorHandling: true }) });
    }

    /**
     * Returns the session token via cookie service - Just an abstraction layer
     */
    public getSessionToken() {
        return this.cookieService.getSessionToken();
    }

    /**
     * Performs the logout operation for the current user
     * Will clear up the cookies that the UI has set up for user management
     *
     * Finally redirects to login page.
     */
    public logout() {
        this.http.post('v1/auth/logout', JSON.stringify({ session: this.getSessionToken() }))
            .subscribe(
            success => {
                this.clearUserSession();
                // Do hard application reload - Clear up any meta states that might exist in services
                window.location.href = '/login';
            },
            error => {
                this.clearUserSession();
                // Do hard application reload - Clear up any meta states that might exist in services
                window.location.href = '/login';
            });
    }

    /**
     * Get the current user object - access cookies.
     */
    public getUser(): User {
        if (this.currentUser === null) {
            throw 'Current user is not loaded';
        }
        return this.currentUser;
    }

    /**
     * Simple clean up work on logout
     */
    public clearUserSession() {
        this.currentUser = null;
        this.currentUserPromise = null;
    }

    public getUsername() {
        return this.getUser().username;
    }

    public getAuthSchemas() {
        return this.http.get('v1/auth/schemes').first().map(res => res.json());
    }

    /**
     * Trigger SAML based login and redirect the page
     */
    public triggerSAMLLogin() {
        let redirectUrl = encodeURIComponent(window.location.href);
        let req = new Observable(observer => {
            this.http.get(`v1/auth/saml/request?redirect_url=${redirectUrl}`)
                .map(res => res.json())
                .subscribe(data => {
                    if (data.request) {
                        window.location.href = data.request;
                    }
                    observer.next(data);
                }, err => {
                    observer.error(err);
                });
        }).first();
        return req;
    }

    public async redirectUnauthenticatedUser() {
        let path = this.locationStrategy.path();
        let isLoginPage = path.indexOf('login') > -1;
        let isTimelinePage = path.indexOf('timeline') > -1;
        let isAnonymousUserSupported = await this.isAnonymousUserSupported();
        if (isAnonymousUserSupported && !isTimelinePage) {
            this.router.navigateByUrl('/app/timeline');
        } else if (!isAnonymousUserSupported && !isLoginPage) {
            this.cookieService.clearSessionCookies();
            this.router.navigateByUrl('/login/' + encodeURIComponent(path));
        }
    }

    private async isAnonymousUserSupported(): Promise<boolean> {
        try {
            await this.http.get('v1/branches?limit=1', { headers: new AxHeaders({ noErrorHandling: true }) }).toPromise();
            return true;
        } catch (e) {
            return false;
        }
    }
}
