import { Component, OnInit } from '@angular/core';
import { AuthenticationService, AuthorizationService, ToolService } from '../../../services';

@Component({
    templateUrl: './introduction.html',
    selector: 'ax-introduction',
    styles: [ require('./introduction.scss') ],
})
export class IntroductionComponent implements OnInit {
    public configureNext = false;
    constructor(
        private authenticationService: AuthenticationService,
        private toolService: ToolService,
        private authorizationService: AuthorizationService) {}

    public ngOnInit() {
        this.authenticationService.getCurrentUser().then(user => {
            if (user.isAdmin() || user.isSuperAdmin()) {
                this.toolService.isScmConfigured().then(isConfigured => {
                    this.configureNext = !isConfigured;
                });
            } else {
                this.configureNext = false;
            }
        });
    }

    public completeIntroduction(route?: any[]) {
        this.authorizationService.completeIntroduction(route);
    }
}
