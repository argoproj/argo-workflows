import { Component, OnInit, OnDestroy } from '@angular/core';
import { Router, ActivatedRoute } from '@angular/router';
import { Subscription } from 'rxjs';

import { User } from '../../../model';
import { UsersService, AuthenticationService, SharedService } from '../../../services';
import { LayoutSettings, HasLayoutSettings } from '../../layout';
import { UserUtils } from '../user-utils';

@Component({
    selector: 'ax-user-profile',
    templateUrl: './user-profile.html',
    styles: [ require('./user-profile.scss') ],
})
export class UserProfileComponent implements OnInit, OnDestroy, LayoutSettings, HasLayoutSettings {

    public editUser = false;
    public changePasswordPanel = false;

    private user: User = new User();
    private username = '';
    private isCurrentLoggedInUser = false;
    private subscriptions: Subscription[] = [];

    constructor(
        private router: Router,
        private usersService: UsersService,
        private authenticationService: AuthenticationService,
        private activatedRoute: ActivatedRoute,
        private utils: UserUtils,
        private sharedService: SharedService) {
    }

    public get layoutSettings(): LayoutSettings {
        return {
            pageTitle: 'Profile',
            breadcrumb: [{
                title: 'All Users',
                routerLink: [`/app/user-management/overview`]
            }, {
                title: this.username,
                routerLink: null,
            }],
            globalAddActionMenu: this.globalAddActionMenu
        };
    }

    public ngOnInit() {
        this.subscriptions.push(this.utils.onUserActionExecuted.subscribe(() => this.loadUser()));
        this.activatedRoute.params.subscribe(params => {
            let username = params['username'];
            if (this.username !== username) {
                this.username = username;
                this.loadUser();
            }
            this.editUser = params['edit'] === 'true';
        });
    }

    public ngOnDestroy() {
        this.subscriptions.forEach(item => item.unsubscribe());
        this.subscriptions = [];
    }

    get globalAddActionMenu() {
        return this.utils.getActionMenu(this.user, item => this.editUserProfile(item.username));
    }

    public editUserProfile(username: string) {
        this.router.navigate([{ edit: 'true' }], { relativeTo: this.activatedRoute });
    }

    public closeEditPanel() {
        this.router.navigate([{ edit: 'false' }], { relativeTo: this.activatedRoute });
    }

    public changePassword() {
        this.changePasswordPanel = true;
    }

    public closeChangePasswordPanel() {
        this.changePasswordPanel = false;
        this.closeEditPanel();
    }

    private async loadUser() {
        let emitToLayout = !this.user.id;
        await this.usersService.getUser(this.username).subscribe(result => {
            this.user = result;
            if (this.username === this.authenticationService.getUsername()) {
                this.isCurrentLoggedInUser = true;
            }
            if (emitToLayout) {
                this.sharedService.updateSource.next(this.layoutSettings);
            }
        });
    }
}
