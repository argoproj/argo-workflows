import { Component, Input, EventEmitter, Output } from '@angular/core';
import { User } from '../../../model';
import { FormControl } from '@angular/forms';
import { FormGroup } from '@angular/forms';
import { AuthenticationService, NotificationService, UsersService } from '../../../services';

@Component({
    selector: 'ax-edit-user',
    templateUrl: './edit-user.html',
    styles: [ require('./edit-user.scss') ],
})
export class EditUserComponent {

    public userForm: FormGroup;
    public _user: User;
    public canChangePassword: boolean;

    @Input()
    public set user(user: User) {
        this._user = user;
        if (user) {
            this.userForm.controls[ 'first_name' ].setValue(user.first_name);
            this.userForm.controls[ 'last_name' ].setValue(user.last_name);
            this.userForm.controls[ 'email' ].setValue(user.username);

            let isSamlOnly = (this._user.auth_schemes || []).indexOf('native') === -1;
            this.canChangePassword = this._user.username === this.authenticationService.getUsername() && !isSamlOnly;
        }
    };

    @Input()
    public show: boolean;

    @Output()
    public onClose: EventEmitter<null> = new EventEmitter();

    @Output()
    public onChangePassword: EventEmitter<null> = new EventEmitter;

    constructor(private usersService: UsersService,
                private notificationService: NotificationService,
                private authenticationService: AuthenticationService) {
        this.userForm = new FormGroup({
            first_name: new FormControl(''),
            last_name: new FormControl(''),
            email: new FormControl({ value: '', disabled: true }),
        });
    }

    public close() {
        this.onClose.emit();
    }

    public update() {
        if (this.userForm.valid) {
            this._user.first_name = this.userForm.controls[ 'first_name' ].value;
            this._user.last_name = this.userForm.controls[ 'last_name' ].value;
            this.usersService.updateUser(this._user).subscribe(result => {
                this.notificationService.showNotification.emit(
                    { message: `User information was successfully updated.` });
            });

            this.close();
        }
    }

    public changePassword() {
        this.onChangePassword.emit();
    }
}
