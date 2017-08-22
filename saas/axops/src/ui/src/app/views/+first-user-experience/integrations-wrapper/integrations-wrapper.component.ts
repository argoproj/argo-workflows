import { Component, OnDestroy } from '@angular/core';
import { Router } from '@angular/router';
import { Subscription } from 'rxjs';

import { ToolService, AuthorizationService } from '../../../services';

@Component({
    selector: 'ax-integrations-wrapper',
    templateUrl: './integrations-wrapper.html',
    styles: [ require('./integrations-wrapper.scss'), require('../first-user-experience.scss') ],
})
export class IntegrationsWrapperComponent implements OnDestroy {

    public isScmConfigured = false;

    private subscription: Subscription;

    constructor(private toolService: ToolService, private authorizationService: AuthorizationService, private router: Router) {
        this.checkIsScmConfigured();
        this.subscription = this.toolService.onToolsChanged.subscribe(this.checkIsScmConfigured.bind(this));
    }

    public ngOnDestroy() {
        this.subscription.unsubscribe();
    }

    public completeIntroduction(force = false) {
        if (this.isScmConfigured || force) {
            this.authorizationService.completeIntroduction();
        }
    }

    private checkIsScmConfigured() {
        this.toolService.isScmConfigured().then(isConfigured => {
            this.isScmConfigured = isConfigured;
        });
    }
}
