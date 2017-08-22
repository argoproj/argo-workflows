import { Directive, ViewContainerRef, TemplateRef } from '@angular/core';

import { AuthenticationService } from '../../services';

@Directive({
    selector: '[axIfAuthenticated]',
})
export class IfAuthenticatedDirective {
    constructor(authorizationService: AuthenticationService, viewContainer: ViewContainerRef, templateRef: TemplateRef<Object>) {
        authorizationService.getCurrentUser().then(user => {
            if (!user.anonymous) {
                viewContainer.createEmbeddedView(templateRef);
            }
        });
    }
}
