import { Component, OnInit, OnDestroy } from '@angular/core';
import { Router, ActivatedRoute } from '@angular/router';
import { BehaviorSubject, Subscription } from 'rxjs';

import { Group, User, SystemRequestType } from '../../model';
import { GroupService, UsersService, SystemRequestService } from '../../services';
import { HasLayoutSettings, LayoutSettings } from '../layout';
import { GlobalSearchSetting, LOCAL_SEARCH_CATEGORIES, ViewUtils } from '../../common';

import { UserUtils } from './user-utils';

interface Invitation {
    firstName: string;
    lastName: string;
    email: string;
    expiry: number;
}

@Component({
    selector: 'ax-manage',
    templateUrl: './manage-users.html',
    styles: [ require('./manage.scss') ],
})
export class ManageUsersComponent implements OnInit, OnDestroy, LayoutSettings, HasLayoutSettings {

    public searchString: string = '';
    public state: number = null;
    public loading: boolean = false;
    public currentView: string;
    public hasTabs: boolean = true;
    public editedUser: User;
    public changePasswordPanel: boolean;
    public showInvitePanel: boolean = false;

    private users: User[] = [];
    private invitations: Invitation[] = [];
    private userGroups: string[];
    private subscriptions: Subscription[] = [];

    constructor(private route: ActivatedRoute,
            private router: Router,
            private usersService: UsersService,
            private groupService: GroupService,
            private systemRequestService: SystemRequestService,
            private utils: UserUtils) {
    }

    get layoutSettings(): LayoutSettings {
        return this;
    }

    get pageTitle(): string {
        return 'Users';
    };

    public globalSearch: BehaviorSubject<GlobalSearchSetting> = new BehaviorSubject<GlobalSearchSetting>({
        suppressBackRoute: false,
        keepOpen: false,
        searchCategory: LOCAL_SEARCH_CATEGORIES.USERS.name,
        searchString: this.searchString,
        applyLocalSearchQuery: (searchString) => {
            this.searchString = searchString;
            this.router.navigate(['app/user-management/overview', this.getRouteParams()]);
        },
    });

    get breadcrumb(): { title: string, routerLink?: any[] }[] {
        return [{
            title: 'All Users',
            routerLink: null,
        }];
    };

    public toolbarFilters = {
        data: [],
        model: [],
        onChange: (data) => {
            this.searchString = null;
            this.state = data.length === 1 ? Number(data[0]) : null;
            this.router.navigate(['app/user-management/overview', this.getRouteParams()]);
        }
    };

    public async ngOnInit() {
        this.subscriptions.push(this.utils.onUserActionExecuted.subscribe(() => this.getUsers()));
        this.route.params.subscribe(params => {
            let showInvitePanel = params['invitePanel'] === 'true';
            if (showInvitePanel !== this.showInvitePanel) {
                this.showInvitePanel = showInvitePanel;
                return;
            }

            let currentlyEditedUserName = this.editedUser && this.editedUser.username || '';
            let editedUser = params['editedUser'] || '';
            if (currentlyEditedUserName !== editedUser) {
                if (editedUser) {
                     this.usersService.getUser(editedUser).subscribe(item => {
                         this.editedUser = item;
                     });
                } else {
                    this.editedUser = null;
                }
                return;
            }

            this.searchString = params['search'] ? decodeURIComponent(params['search']) : null;
            this.state = params['state'] ? Number(decodeURIComponent(params['state'])) : null;
            if (params['state'] && this.toolbarFilters.model.length === 0) {
                this.toolbarFilters.model = [decodeURIComponent(params['state'])];
            }

            if (this.searchString) {
                this.globalSearch.value.searchString = this.searchString;
                this.globalSearch.value.keepOpen = true;
            }
            this.currentView = params['view'] || 'users';
            if (this.currentView === 'users') {
                this.getUsers();
                this.toolbarFilters.data = [{
                    value: '2',
                    name: 'Active',
                    icon: { color: 'running' },
                }, {
                    value: '3',
                    name: 'Inactive',
                    icon: { color: 'success' },
                }];
            } else {
                this.getInvitations();
                this.toolbarFilters.data = [];
            }
        });
    }

    public ngOnDestroy() {
        this.subscriptions.forEach(item => item.unsubscribe());
        this.subscriptions = [];
    }

    public getUserMenu(user: User) {
        return this.utils.getActionMenu(user, item => this.editUserProfile(item.username));
    }

    changeView(view: string) {
        this.router.navigate(['.', Object.assign(this.getRouteParams(), { view })], { relativeTo: this.route });
    }

    globalAddAction() {
        this.router.navigate(['app/user-management/overview', this.getRouteParams({ invitePanel: 'true' })]);
    }

    applySearchQuery(searchString) {
        this.searchString = searchString;
        this.router.navigate(['app/user-management/overview', this.getRouteParams()]);
    }

    closeInvitePanel() {
        this.router.navigate(['app/user-management/overview', this.getRouteParams({ invitePanel: null })]);
    }

    display(state: number, e?) {
        this.searchString = null;
        this.state = state;
        this.router.navigate(['/app/user-management/overview', this.getRouteParams()]);
    }

    displayGroups(e) {
        this.display(null, e);
        this.groupService.getGroups().subscribe(result => {
            this.userGroups = Group.getGroupList(result.data);
        });
    }

    getUsers(): void {
        let params = {
            state: this.state,
            search: this.searchString,
        };
        this.loading = true;
        this.usersService.getUsers(params, true).subscribe(result => {
            this.users = result.data;
            this.loading = false;
        }, err => {
            this.loading = false;
        });
    }

    getInvitations() {
        this.loading = true;
        this.invitations = [];
        this.systemRequestService.getSystemRequests({ type: SystemRequestType.UserInvite }, true).then(requests => {
            this.loading = false;
            this.invitations = requests.map(item => ({
                firstName: item.data.firstName,
                lastName: item.data.lastName,
                email: item.target,
                expiry: item.expiry
            }));
        }, err => {
            this.loading = false;
        });
    }

    editUserProfile(username: string) {
        this.router.navigate(['app/user-management/overview', this.getRouteParams({ editedUser: username })]);
    }

    public closeEditPanel() {
        this.router.navigate(['app/user-management/overview', this.getRouteParams({ editedUser: null })]);
    }

    public changePassword() {
        this.changePasswordPanel = true;
    }

    public closeChangePasswordPanel() {
        this.changePasswordPanel = false;
        this.closeEditPanel();
    }

    private getRouteParams(updatedParams?) {
        let params = {};
        if (this.searchString) {
            params['search'] = this.searchString;
        }
        if (this.state) {
            params['state'] = this.state.toString();
        }
        return ViewUtils.sanitizeRouteParams(params, updatedParams);
    }
}
