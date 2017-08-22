import { Component, OnInit } from '@angular/core';
import { Router, ActivatedRoute } from '@angular/router';
import { FormGroup, FormControl, Validators } from '@angular/forms';

import { CustomRegex } from '../../../common/customValidators/CustomRegex';
import { CustomValidators } from '../../../common/customValidators/CustomValidators';
import { UsersService, AuthenticationService, AuthorizationService } from '../../../services';

@Component({
    selector: 'ax-sign-up',
    templateUrl: './sign-up.html',
})
export class SignUpComponent implements OnInit {
    signupForm: FormGroup;
    token: string = '';
    avatar: string = '';
    email: string = '';

    constructor(private router: Router,
                private route: ActivatedRoute,
                private usersService: UsersService,
                private authenticationService: AuthenticationService,
                private authorizationService: AuthorizationService) {
    }

    ngOnInit() {
        this.signupForm = new FormGroup({
            first_name: new FormControl(''),
            last_name: new FormControl(''),
            password: new FormControl('', Validators.compose([
                Validators.required, Validators.pattern(CustomRegex.password), Validators.minLength(8)])),
            repeat_password: new FormControl(''),
            username: new FormControl('', Validators.compose([
                Validators.required, Validators.pattern(CustomRegex.email)]))
        }, CustomValidators.matchingPasswords);

        this.route.params.subscribe(params => {
            this.token = params['token'] ? decodeURIComponent(params['token']) : null;
            this.email = params['email'] ? atob(params['email']) : '';
            this.signupForm.controls['username'].setValue(this.email);
        });
    }

    get isSingleUser() {
        return !!this.email;
    }

    createAccount(form) {
        if (form.valid) {
            this.usersService.registerUser({
                first_name: form.value.first_name,
                last_name: form.value.last_name,
                password: form.value.password,
                username: form.value.username,
            }, this.token).subscribe(success => {
                if (this.isSingleUser) {
                    this.authenticationService.login(this.email, form.value.password).subscribe(_ => {
                        this.authorizationService.redirectIfSessionExists();
                    });
                } else {
                    this.router.navigate(['/setup/confirm']);
                }
            });
        }
    }
}
