import { Component, OnInit } from '@angular/core';
import { FormGroup, FormControl, Validators } from '@angular/forms';
import { ActivatedRoute, Router } from '@angular/router';

import { UsersService, NotificationService } from '../../services';
import { CustomValidators } from '../../common/customValidators/CustomValidators';
import { CustomRegex } from '../../common/customValidators/CustomRegex';


@Component({
    selector: 'ax-reset-password',
    templateUrl: './reset-password.html',
})
export class ResetPasswordComponent implements OnInit {
    public submitted: boolean;

    private resetPasswordForm: FormGroup;
    private username: string = '';
    private token: string = '';

    constructor(private route: ActivatedRoute,
                private router: Router,
                private usersService: UsersService,
                private notificationService: NotificationService) {
    }

    ngOnInit() {
        this.route.params.subscribe(params => {
            this.username = params['username'] ? atob(params['username']) : null;
            this.token = params['token'] ? decodeURIComponent(params['token']) : null;
        });

        this.resetPasswordForm = new FormGroup({
            new_password: new FormControl('',
                Validators.compose([Validators.required, Validators.pattern(CustomRegex.password), Validators.minLength(8)])),
            confirm_password: new FormControl('', Validators.required)
        }, CustomValidators.matchProperties('new_password', 'confirm_password'));
    }

    resetPassword(resetPasswordForm: FormGroup) {
        this.submitted = true;
        if (resetPasswordForm.valid) {
            this.usersService.resetPassword(this.username, this.token, resetPasswordForm.value).subscribe(() => {
                this.router.navigate(['reset-password/confirm']);
                this.notificationService.showNotification.emit({message: `Reset password was successfully.`});
            });
        }
    }
}
