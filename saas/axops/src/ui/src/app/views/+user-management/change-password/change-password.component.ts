import { Component, Input, EventEmitter, Output, OnInit } from '@angular/core';
import { User } from '../../../model';
import { FormControl, Validators } from '@angular/forms';
import { FormGroup } from '@angular/forms';
import { NotificationService, UsersService } from '../../../services';
import { CustomRegex } from '../../../common/customValidators/CustomRegex';
import { CustomValidators } from '../../../common/customValidators/CustomValidators';


@Component({
    selector: 'ax-change-password',
    templateUrl: './change-password.html',
    styles: [ require('./change-password.scss') ],
})
export class ChangePasswordComponent implements OnInit {

    public changePasswordForm: FormGroup;
    public submitted: boolean;

    @Input()
    public show: boolean;

    @Input()
    public user: User;

    @Output()
    public onClose: EventEmitter<null> = new EventEmitter();

    constructor(private usersService: UsersService,
                private notificationService: NotificationService) {
    }

    public ngOnInit() {
        this.changePasswordForm = new FormGroup({
            old_password: new FormControl(''),
            new_password: new FormControl('', [Validators.required, Validators.pattern(CustomRegex.password), Validators.minLength(8)]),
            confirm_password: new FormControl('', Validators.required)
        }, CustomValidators.matchProperties('new_password', 'confirm_password'));
    }

    public close() {
        this.onClose.emit();
    }

    public update() {
        this.submitted = true;
        if (this.changePasswordForm.valid) {
            let changePassword: { 'old_password': string, 'new_password': string, 'confirm_password': string } = this.changePasswordForm.value;

            this.usersService.updatePassword(this.user.username, changePassword).subscribe(() => {
                this.notificationService.showNotification.emit({message: `Password was successfully updated.`});
                this.close();
            });
        }
    }
}
