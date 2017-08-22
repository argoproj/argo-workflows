import { Component, Input, Output, EventEmitter } from '@angular/core';

import { UsersService } from '../../services';
import { User } from '../../model';
import { SortOperations } from '../sortOperations/sortOperations';

export class GroupedUsers {
    name: string;
    checked: boolean;
    users: { checked: boolean, name: string, email: string, display?: boolean }[];
    isExpanded?: boolean;
    display?: boolean;
}

@Component({
    selector: 'ax-users-selector-panel',
    templateUrl: './users-selector-panel.html',
    styles: [require('./users-selector-panel.scss')],
})
export class UsersSelectorPanelComponent {
    @Input()
    public selectedUsers: string[];

    @Input()
    public axUsersList: User[];

    @Input()
    set show(val: boolean) {
        this.isPanelVisible = val;
        if (this.users.length === 0 && this.isPanelVisible) {
            if (this.axUsersList) {
                this.prepareUsersToDisplay(this.axUsersList);
            } else {
                this.getUsers();
            }
        } else if (this.users.length) {
            this.usersToDisplay = JSON.parse(JSON.stringify(this.users));
            this.selectUsers();
        }
    }

    @Output()
    public onChange: EventEmitter<string[]> = new EventEmitter();

    @Output()
    public onClose: EventEmitter<any> = new EventEmitter();

    public isPanelVisible: boolean = false;
    public usersToDisplay: GroupedUsers[] = [];
    public getUsersLoader: boolean = false;
    public searchedUser: string;
    private users: GroupedUsers[] = [];

    constructor(private usersService: UsersService) {
    }

    public add() {
        let groupAndUserList: string[] = [];
        this.usersToDisplay.forEach(group => {
           if (group.checked) {
               groupAndUserList.push(`${group.name}@group`);
           }

           group.users.forEach(user => {
               if (user.checked) {
                   groupAndUserList.push(user.email);
               }
           });
        });

        this.onChange.emit(groupAndUserList);
        this.closeUserSelectorSlidingPanel();
    }

    public closeUserSelectorSlidingPanel() {
        this.onClose.emit();
    }

    public changed(searchString: string) {
        this.usersToDisplay.forEach(group => {
            group.display = group.name.toLowerCase().indexOf(searchString.toLowerCase()) !== -1;

            group.users.forEach(user => {
                user.display = user.name.toLowerCase().indexOf(searchString.toLowerCase()) !== -1 || user.email.toLowerCase().indexOf(searchString.toLowerCase()) !== -1;
            });
        });
    }

    private getUsers() {
        this.getUsersLoader = true;
        this.usersService.getUsers({}, true).toPromise().then(res => {
            this.prepareUsersToDisplay(SortOperations.sortBy(res.data, 'last_name'));
            this.getUsersLoader = false;
        });
    }

    private prepareUsersToDisplay(users: User[]) {
        users = users.filter(user => user.groups[0] !== 'super_admin'); // do not show super_admin users
        this.users = this.shareToGroups(users);
        this.usersToDisplay = JSON.parse(JSON.stringify(this.users));
        this.selectUsers();
    }

    private shareToGroups(users: User[]): GroupedUsers[] {
        let uniqueListOfGroups: string[] = [];
        let usersSortedByGroup: GroupedUsers[] = [];
        users.forEach((user: User) => {
            if (uniqueListOfGroups.indexOf(user.groups[0]) === -1) {
                uniqueListOfGroups.push(user.groups[0]);
                usersSortedByGroup.push({ name: user.groups[0], users: [], checked: false, isExpanded: true, display: true });
                usersSortedByGroup[uniqueListOfGroups.indexOf(user.groups[0])].users.push({
                    checked: false,
                    name: (user.first_name && user.last_name) ? `${user.first_name} ${user.last_name}` : user.username,
                    email: user.username,
                    display: true });
            } else {
                usersSortedByGroup[uniqueListOfGroups.indexOf(user.groups[0])].users.push({
                    checked: false, name: `${user.first_name} ${user.last_name}`, email: user.username, display: true });
            }
        });

        return usersSortedByGroup;
    }

    private selectUsers() {
        this.usersToDisplay.forEach(group => {
            group.checked = this.selectedUsers.indexOf(`${group.name}@group`) !== -1;

            group.users.forEach(user => {
                user.checked = this.selectedUsers.indexOf(user.email) !== -1;
            });
        });
    }
}
