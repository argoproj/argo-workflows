import { Component, OnInit } from '@angular/core';
import { FormGroup, FormControl, Validators } from '@angular/forms';
import { Router } from '@angular/router';

import { UsersService } from '../../services';
import { CustomValidators } from '../../common/customValidators/CustomValidators';
import { NotificationsService } from 'argo-ui-lib/src/components';

@Component({
    selector: 'ax-forgot-password',
    templateUrl: './forgot-password.html',
})
export class ForgotPasswordComponent implements OnInit {
    public submitted: boolean;
    private resetPasswordForm: FormGroup;

    constructor(private router: Router, private usersService: UsersService, private notificationsService: NotificationsService) {
    }

    ngOnInit() {
        this.resetPasswordForm = new FormGroup({
            email: new FormControl('', [Validators.required, CustomValidators.emailValidator])
        });
    }

    resetPassword(resetPasswordForm: FormGroup) {
        this.submitted = true;
        if (resetPasswordForm.valid) {
            let email = resetPasswordForm.value['email'];
            this.usersService.forgetPassword(email).subscribe(result => {
                this.router.navigate(['forgot-password/confirm']);
            }, err => {
                this.notificationsService.error(`The user with email "${email}" does not exist.`);
            });
        }
    }
}
