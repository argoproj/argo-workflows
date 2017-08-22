import * as _ from 'lodash';
import * as moment from 'moment';
import { Component, OnInit, OnDestroy } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { FormGroup, FormControl, Validators } from '@angular/forms';

import { EventsService, AuthenticationService, AuthorizationService, SystemService , DEFAULT_FWD_URL } from '../../services';

@Component({
    selector: 'ax-login',
    templateUrl: './login.component.html',
    styles: [ require('./login.component.scss') ],
})
export class LoginComponent implements OnInit, OnDestroy {
    public year: string;
    public version: string;
    public submitted: boolean;

    private loginForm: FormGroup;
    private showLoader: boolean = false;
    private isVissibleWrongCredentialsMsg: boolean = false;
    private schemas: any = null;
    private samlLogin: boolean = false;
    // Default value - api will send otherwise
    private samlButtonLabel: string = 'Single Sign On';

    // Url to send the user after login
    private fwdUrl: string = DEFAULT_FWD_URL;

    constructor(public _router: Router,
                private eventsService: EventsService,
                private authenticationService: AuthenticationService,
                private authorizationService: AuthorizationService,
                private activatedRoute: ActivatedRoute,
                private systemService: SystemService) {
    }

    public ngOnInit() {
        this.year = moment().year().toString();
        this.systemService.getVersion().subscribe(info => this.version = info.version.split('-')[0]);

        // Read the url for any forward urls
        this.activatedRoute.params.subscribe(params => {
            this.fwdUrl = params['fwd'] ? decodeURIComponent(params['fwd']) : this.fwdUrl;
        });

        this.authenticationService.getAuthSchemas().subscribe(res => {
            this.schemas = res.data;
            _.each(this.schemas, (item) => {
                if (item.name === 'saml') {
                    this.samlLogin = item.enabled;
                    if (this.samlLogin) {
                        this.samlButtonLabel = item.button_label ? item.button_label : this.samlButtonLabel;
                    }
                }
            });
        });

        this.eventsService.hideNavigation.emit({});

        this.loginForm = new FormGroup({
            password: new FormControl('', Validators.required),
            username: new FormControl('', Validators.required)
        });

        this.loginForm.valueChanges.subscribe((result) => {
            this.isVissibleWrongCredentialsMsg = false;
        });
    }

    /**
     * Post login operations
     */
    postLogin() {
        this.authorizationService.redirectIfSessionExists(this.fwdUrl);
    }

    ngOnDestroy() {
        this.eventsService.showNavigation.emit({});
    }

    /**
     * Login handler.
     */
    login(form) {
        this.submitted = true;
        if (form.valid) {
            this.showLoader = true;
            this.authenticationService.login(
                form.value.username,
                form.value.password
            ).subscribe(success => {
                this.postLogin();
            }, error => {
                this.wrongCredential(error);
            });
        }
    }

    /**
     * Handles errors in login form.
     */
    wrongCredential(error) {
        this.showLoader = false;
        this.isVissibleWrongCredentialsMsg = true;
    }

    loginWithSaml() {
        this.authenticationService.triggerSAMLLogin().subscribe(data => {
            // do nothing
        });
    }

    a() {
        console.log('work');
    }
}
