import { Injectable, EventEmitter } from '@angular/core';
import { DropdownMenuSettings } from 'argo-ui-lib/src/components';

import { AuthenticationService, NotificationService, UsersService, ModalService } from '../../services';
import { User } from '../../model';

export enum UserState {
    Init = 1,
    Active = 2,
    Banned = 3,
    Deleted = -1,
}

@Injectable()
export class UserUtils {

    public onUserActionExecuted = new EventEmitter();

    constructor(
        private authenticationService: AuthenticationService,
        private notificationService: NotificationService,
        private usersService: UsersService,
        private modalService: ModalService) {
    }

    public getActionMenu(user: User, editAction: (user: User) => any): DropdownMenuSettings {
        let items = [{title: 'Edit Profile', action: () => editAction(user), iconName: '' }];
        if (user.username !== this.authenticationService.getUsername()) {
            if (user.state === UserState.Active) {
                items.push({title: 'Deactivate User', action: () => this.deactivate(user.username), iconName: '' });
            }
            if (user.state === UserState.Banned) {
                items.push({title: 'Activate User', action: () => this.activateUser(user.username), iconName: '' });
            }
            if (user.state === UserState.Init || user.state === UserState.Active || user.state === UserState.Banned) {
                items.push({title: 'Delete User', action: () => this.deleteUser(user.username), iconName: '' });
            }
            if (user.state === UserState.Init) {
                items.push({title: 'Re-Send Confirmation Email', action: () => this.reSendConfirmation(user.username), iconName: '' });
            }
            if (user.isAdmin()) {
                items.push({title: 'Revoke Admin Access', action: () => this.revokeAdminAccess(user), iconName: '' });
            }
            if (!user.isAdmin()) {
                items.push({title: 'Grant Admin Access', action: () => this.grantAdminAccess(user), iconName: '' });
            }
        }

        return new DropdownMenuSettings(items, 'fa-ellipsis-v');
    }

    private deactivate(usernameToBan: string): void {
        this.usersService.banUser(usernameToBan).subscribe(result => {
            this.notificationService.showNotification.emit(
                { message: `User ${usernameToBan} has been banned.` });
            this.onUserActionExecuted.emit({});
        });
    }

    private activateUser(usernameToActivate: string): void {
        this.usersService.activateUser(usernameToActivate).subscribe(result => {
            this.notificationService.showNotification.emit(
                { message: `User ${usernameToActivate} has been activated.` });
            this.onUserActionExecuted.emit({});
        });
    }

    /**
     * Retrigger the invitation email
     */
    private reSendConfirmation(username: string): void {
        this.usersService.resendConfirmationEmail(username).subscribe(result => {
            this.notificationService.showNotification.emit(
                { message: `Invitation for ${username} was resend.` });
            this.onUserActionExecuted.emit({});
        });
    }

    /**
     * Archive a user
     */
    private deleteUser(usernameToArchive: string): void {
        this.usersService.archiveUser(usernameToArchive).subscribe(result => {
            this.notificationService.showNotification.emit(
                { message: `User ${usernameToArchive} has been archived.` });
            this.onUserActionExecuted.emit({});
        });
    }

    /**
     * Revoke Admin Group Access
     *
     */
    private revokeAdminAccess(user: User) {
        this.modalService.showModal('Revoke Admin Access', `Do you want to remove admin privileges for the user: “${user.username}“?`)
            .subscribe(result => {
                if (result) {
                    user.removeAdminAccess();
                    this.usersService.updateUser(user).subscribe(() => {
                        this.notificationService.showNotification.emit(
                            { message: `Admin access revoked for user: ${user.username}.` });
                        this.onUserActionExecuted.emit({});
                    });
                }
            });
    }

    /**
     * Grant Admin Group Access
     *
     */
    private grantAdminAccess(user: User) {
        this.modalService.showModal('Grant Admin Access', `Do you want to give admin privileges to the user: “${user.username}“?`)
            .subscribe(result => {
                if (result) {
                    user.giveAdminAccess();
                    this.usersService.updateUser(user).subscribe(() => {
                        this.notificationService.showNotification.emit(
                            { message: `Admin privileges granted for user: ${user.username}.` });
                        this.onUserActionExecuted.emit({});
                    });
                }
            });
    }

}
