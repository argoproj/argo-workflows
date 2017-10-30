import { Injectable, EventEmitter } from '@angular/core';

import { ViewPreferences, User, Branch } from '../model';
import { AuthenticationService } from './authentication.service';
import { BranchService } from './branch.service';

import { Observable } from 'rxjs';

export interface ViewPreferencesChangeInfo {
    previousPreferences: ViewPreferences;
    viewFavoriteUpdated: boolean;
}

@Injectable()
export class ViewPreferencesService {

    public onPreferencesUpdated = new EventEmitter<ViewPreferences & { changeInfo?: ViewPreferencesChangeInfo }>();

    constructor(
        private authenticationService: AuthenticationService,
        private branchService: BranchService) {

        this.authenticationService.onUserUpdated.subscribe((user: User) => {
            this.onPreferencesUpdated.emit(this.deserializePreferences(user.view_preferences));
        });

        this.checkIfAllBranchesExist();
    }

    public getViewPreferences(): Promise<ViewPreferences> {
        return this.authenticationService.getCurrentUser().then(user => this.deserializePreferences(user.view_preferences));
    }

    public getViewPreferencesObservable(): Observable<ViewPreferences & { changeInfo?: { viewFavoriteUpdated: boolean } }> {
        return Observable.merge(this.onPreferencesUpdated, Observable.fromPromise(this.getViewPreferences()));
    }

    public updateViewPreferences(callback: (viewPreferences: ViewPreferences) => any): Promise<boolean> {
        return this.getViewPreferences().then(viewPreferences => {
            let before = JSON.stringify(viewPreferences);
            callback(viewPreferences);
            let after = JSON.stringify(viewPreferences);
            if (before !== after) {
                return this.updateUserPrefences(viewPreferences).then(() => true);
            } else {
                return false;
            }
        });
    }

    private checkIfAllBranchesExist() {
        this.getViewPreferences().then((v: ViewPreferences) => {
            if (v.favouriteBranches.length > 0) {
                this.branchService.getBranchesAsync({ name: v.favouriteBranches.map(item => `^${item.name}$`).toString().replace(/,/g, '|') }).toPromise().then(res => {
                    // Apply repo name filtering
                    let keyToBranch = new Map<string, Branch>();
                    res.data.forEach(item => keyToBranch.set(`${item.repo}:${item.name}`, item));
                    v.favouriteBranches = v.favouriteBranches.map(item => keyToBranch.get(`${item.repo}:${item.name}`)).filter(item => !!item);

                    this.updateViewPreferences(viewPreferences => viewPreferences.favouriteBranches = v.favouriteBranches);
                });
            }
        });
    }

    private async updateUserPrefences(preferences: ViewPreferences) {
        let previousPreferences: ViewPreferences;
        await this.authenticationService.updateCurrentUser(user => {
            previousPreferences = this.deserializePreferences(user.view_preferences);
            user.view_preferences = this.serializePreferences(preferences);
        }, true);
        let changeInfo: ViewPreferencesChangeInfo = {
            previousPreferences,
            viewFavoriteUpdated:
                previousPreferences.filterState.branches !== preferences.filterState.branches ||
                previousPreferences.favouriteBranches.map(branch => branch.id).join(',') !== preferences.favouriteBranches.map(branch => branch.id).join(',')
        };
        this.onPreferencesUpdated.emit(Object.assign(preferences, { changeInfo }));
    }

    private serializePreferences(preferences: ViewPreferences) {
        let viewPreferences = {};
        for (let key in preferences) {
            if (preferences.hasOwnProperty(key)) {
                viewPreferences[key] = JSON.stringify(preferences[key]);
            }
        }
        return viewPreferences;
    }

    private deserializePreferences(view_preferences): ViewPreferences {
        let viewPreferences = {
            favouriteBranches: [],
            isIntroductionCompleted: false,
            playgroundTask: null,
            mostRecentNotificationsViewTime: 0,
            filterState: { branches: <'all' | 'my'> 'all'},
            filterStateInPages: {},
            firstJobFeedbackStatus: null,
        };
        let userPrefs = view_preferences || {};
        for (let key in view_preferences) {
            if (view_preferences.hasOwnProperty(key)) {
                let prefJson = userPrefs[key];
                try {
                    if (prefJson) {
                        viewPreferences[key] = JSON.parse(prefJson);
                    }
                } catch (e) {
                    // Ignore invalid user preference
                }
            }
        }
        return viewPreferences;
    }
}
