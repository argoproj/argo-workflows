import { Directive, ViewContainerRef, TemplateRef, Input } from '@angular/core';

import { AuthenticationService } from '../../services';

@Directive({
    selector: '[axIfUserIsInGroup]',
})
export class IfUserIsInGroupDirective {
    constructor(private authorizationService: AuthenticationService, private viewContainer: ViewContainerRef, private templateRef: TemplateRef<Object>) {
    }

    @Input() set axIfUserIsInGroup(value: string | string[]) {
        this.authorizationService.getCurrentUser().then(user => {
            if (user.anonymous) {
                this.viewContainer.clear();
                return;
            }

            let userGroups = user.groups || [];
            let groups = typeof value === 'string' ? [value] : <string[]> value;
            let isUserInRole = !!groups.find(group => userGroups.indexOf(group) > - 1);
            if (isUserInRole) {
                this.viewContainer.createEmbeddedView(this.templateRef);
            } else {
                this.viewContainer.clear();
            }
        });
    }
}
