import * as _ from 'lodash';
import { Injectable } from '@angular/core';
import { Http, URLSearchParams, Headers } from '@angular/http';
import { Observable } from 'rxjs';

import { User, Label } from '../model';
import { AxHeaders } from './headers';

@Injectable()
export class UsersService {
    constructor(private http: Http) {
    }

    /**
     * Get users in the account
     * Filter based on state or a search string
     */
    getUsers(params: {
        state?: number,
        search?: string,
    }, hideLoader?: boolean): Observable<{ data: User[] }> {
        let customHeader = new Headers();
        let search = new URLSearchParams();
        if (hideLoader) {
            customHeader.append('isUpdated', hideLoader.toString());
        }
        if (params.state) {
            search.set('state', encodeURIComponent(params.state.toString()));
        }
        if (params.search) {
            search.set('search', '~' + encodeURIComponent(params.search.toString()));
        }

        return this.http.get(`v1/users?${search.toString()}`, { headers: customHeader }).map((res) => {
            let result = res.json(), userData: User[] = [];
            // Cast data to User Object
            _.forEach(result.data, (user) => {
                userData.push(new User(user));
            });
            result.data = userData;
            return result;
        });
    }

    /**
     * Get user profile with the username
     */
    getUser(username: string): Observable<User> {
        return this.http.get(`v1/users/${username}`).map(res => new User(res.json()));
    }

    /**
     * Ban a particular user
     */
    banUser(username: string) {
        return this.http.put(`v1/users/${username}/ban`, {}).map(res => res.json());
    }

    /**
     * Activate a user profile
     */
    activateUser(username: string) {
        return this.http.put(`v1/users/${username}/activate`, {}).map(res => res.json());
    }

    /**
     * Resent the activation email (link) for a particular user
     */
    resendConfirmationEmail(username: string) {
        return this.http.post(`v1/users/${username}/resend_confirm`, {}).map(res => res.json());
    }

    /**
     * Trigger Forgot password functionality.
     * This will trigger an email being sent to the user
     */
    forgetPassword(username: string) {
        return this.http.post(`v1/users/${username}/forget_password`, {}, { headers: new AxHeaders({ noErrorHandling: true }) }).map(res => res.json());
    }

    /**
     * Reset a password
     * User gets a token to help achieve this via an email link
     */
    resetPassword(username: string, token: string, passwords: { new_password: string, old_password: string }) {
        return this.http.put(`v1/users/${username}/reset_password/${token}`, JSON.stringify(passwords)).map(res => res.json());
    }

    /**
     * Delete (Archive) a user profile. User is deactivated from the system
     */
    archiveUser(username: string) {
        return this.http.delete(`v1/users/${username}`).map(res => res.json());
    }

    /**
     * Trigger a registeration functionality for a user.
     * Use the token to create a profile.
     */
    registerUser(user: {first_name: string, last_name: string, password: string, username: string}, token: string) {
        return this.http.post(`v1/users/${user.username}/register/${token}`, JSON.stringify(user)).map(res => res.json());
    }

    saveUserAsync(user: { username: string }) {
        return this.http.post('v1/user', JSON.stringify(user)).map(res => res.json());
    }

    updateUser(user: User, hideLoader?: boolean) {
        let customHeader = new Headers();
        if (hideLoader) {
            customHeader.append('isUpdated', hideLoader.toString());
        }
        return this.http.put(`v1/users/${user.username}`, JSON.stringify(user), {headers: customHeader}).map(res => res.json());
    }

    /**
     * Change password for a user profile
     */
    updatePassword(username: string, changePassword: { 'old_password': string, 'new_password': string, 'confirm_password': string }) {
        return this.http.put(`v1/users/${username}/change_password`, JSON.stringify(changePassword)).map(res => res.json());
    }

    loadUsers(): Observable<{ data: User[] }> {
        return this.http.get('v1/users').map(res => res.json());
    }
    /**
     * Load labels of type user
     */
    loadUserLabels(): Observable<{ data: Label[] }> {
        return this.http.get('v1/labels').map(res => res.json());
    }

    /**
     * Admin use this api to send invitations to users or distribution lists
     */
    inviteUser(username: string, group: string, isSingleUser: boolean, firstName: string, lastName: string) {
        let search = new URLSearchParams();
        search.set('group', group);
        if (firstName) {
            search.set('first_name', firstName);
        }
        if (lastName) {
            search.set('last_name', lastName);
        }
        if (isSingleUser) {
            search.set('single_user', 'true');
        }
        return this.http.post(`v1/users/${username}/invite?${search.toString()}`, {}).map(res => res.json());
    }

}
