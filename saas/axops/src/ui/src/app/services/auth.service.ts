import * as _ from 'lodash';
import { Injectable } from '@angular/core';
import { Observable } from 'rxjs/Rx';
import { Router, CanActivate, ActivatedRouteSnapshot, RouterStateSnapshot } from '@angular/router';
import { User } from '../model';
import { permissions } from '../permissions';
import { AuthenticationService } from './authentication.service';
import { ViewPreferencesService } from './view-preferences.service';
import { SystemService } from './system.service';

export const DEFAULT_FWD_URL = 'app/timeline';

@Injectable()
export class AuthorizationService {
    /**
     * Permissions data from permissions.ts gets processed and persisted here
     */
    private perms = {};

    // Url to send the user after login
    private fwdUrl: string = DEFAULT_FWD_URL;

    constructor(
        private router: Router,
        private authenticationService: AuthenticationService,
        private viewPreferencesService: ViewPreferencesService) {
        _.each(permissions, perm => {
            if (perm && perm['path']) {
                let p = perm['path'];
                if (p.indexOf('*') > -1) {
                    this.perms[p.split('*')[0]] = perm;
                    perm['exact'] = false;
                } else {
                    this.perms[p] = perm;
                    perm['exact'] = true;
                }
            }
        });
    }

    /**
     * This method checks for a path to be accessible for current user or not.
     */
    hasAccess(route: ActivatedRouteSnapshot, state: RouterStateSnapshot): Promise<boolean> {
        return this.authenticationService.getCurrentUser().then(currentUser => this.checkIfHasAccess(currentUser, state));
    }

    checkIfHasAccess(currentUser: User, state: RouterStateSnapshot) {
        let flag = false,
            matchFound = false,
            currentUrl = state.url,
            matchKey = '',
            matchingGroups = [];
        if (currentUser) {
            for (let key in this.perms) {
                if (this.perms.hasOwnProperty(key)) {
                    if (currentUrl.startsWith(key)) {
                        flag = true;
                        matchFound = true;
                        // value of key when match was flagged!
                        matchKey = key;
                        if (this.perms[key].exact && currentUrl !== key) {
                            // if we are looking for exact match and we dont have it
                            // make sure iteration continues
                            flag = false;
                        }

                        // If no permissions are defined - we assume user has access.
                        matchingGroups = currentUser.groups;
                        if (this.perms[key].permission.length > 0) {
                            matchingGroups = _.intersection(this.perms[key].permission, currentUser.groups);
                        }

                        if (flag && matchingGroups.length === 0) {
                            // If no matchingGroups exist, then block access
                            flag = false;
                        }

                    }

                    if (flag) {
                        break;
                    }
                }
            }
        }

        // IF no match is found -  we will still allow redirection to requested path
        if (!matchFound) {
            flag = true;
        }
        if (!flag) {
            console.log('Access check failed for url:',
                state.url, 'user-perms:', currentUser.groups, 'match key:', matchKey, 'permissions:', this.perms[matchKey]);
            this.redirectTo401();
        }
        return flag;
    }

    hasSession(route: ActivatedRouteSnapshot, state: RouterStateSnapshot): Observable<boolean> {
        return new Observable<boolean>(observer => {
            // Load the user
            this.authenticationService.getCurrentUser().then((success) => {
                // Successfuly loaded the user into the app based on session
                observer.next(true);
            }).catch((error) => {
                // Unable to load user - hence no session is existing
                this.authenticationService.redirectUnauthenticatedUser();
                observer.next(false);
            });
        });
    }

    redirectTo401() {
        this.router.navigateByUrl('/error/401');
    }

    redirectIfSessionExists(fwdUrl = '') {
        this.router.navigateByUrl(fwdUrl || this.fwdUrl);
    }

    completeIntroduction(route?: any[]) {
        this.viewPreferencesService.updateViewPreferences(preferences => {
            preferences.isIntroductionCompleted = true;
            preferences.playgroundTask = null;
            this.router.navigate(route || [this.fwdUrl], { queryParams: { tutorial: true } });
        });
    }

    hasNoSession(route: ActivatedRouteSnapshot, state: RouterStateSnapshot): boolean {
        let flag = true;
        if (this.authenticationService.getSessionToken()) {
            this.redirectIfSessionExists(decodeURIComponent(route.params['fwd'] ? route.params['fwd'] : ''));
            flag = false;
        }
        return flag;
    }
}

/**
 * This is a joint filter for checking session and access in a synchronized manner.
 * HasAccess conditions will be invoked only if HasSession is successful
 */
@Injectable()
export class UserAccessControl implements CanActivate {
    constructor(
        private authService: AuthorizationService,
        private viewPreferencesService: ViewPreferencesService,
        private router: Router,
        private systemService: SystemService) { }
    canActivate(route: ActivatedRouteSnapshot,
        state: RouterStateSnapshot): Observable<boolean> | boolean {
        return new Observable<boolean>(observer => {
            this.authService.hasSession(route, state).first().toPromise().then(success => {
                if (success) {
                    this.authService.hasAccess(route, state).then(hasAccess => {
                        if (hasAccess) {
                            this.viewPreferencesService.getViewPreferences().then(preferences => {
                                if (!preferences.isIntroductionCompleted && !preferences.playgroundTask) {
                                    this.systemService.isPlayground().then(isPlayground => {
                                        this.router.navigateByUrl(isPlayground ? '/fue/playground' : '/fue');
                                        observer.next(false);
                                    });
                                } else {
                                    observer.next(hasAccess);
                                }
                            });
                        }
                    });
                } else {
                    observer.next(success);
                }
            }, error => {
                observer.next(false);
            });
        }).first();
    }
}


/**
 * This Authorization filter will be applied to urls that need to be accessed only if session exists
 * If session does not exist - This will redirect to login screen
 * Example: all routes like /app/*
 */
@Injectable()
export class HasNoSession implements CanActivate {
    constructor(private authService: AuthorizationService) { }
    canActivate(route: ActivatedRouteSnapshot, state: RouterStateSnapshot): Observable<boolean> | boolean {
        return this.authService.hasNoSession(route, state);
    }
}
