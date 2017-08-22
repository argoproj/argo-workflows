import { Directive, ViewContainerRef, TemplateRef } from '@angular/core';

import { AuthenticationService } from '../../services';

@Directive({
    selector: '[axIfAnonymous]',
})
export class IfAnonymousDirective {
    constructor(authorizationService: AuthenticationService, viewContainer: ViewContainerRef, templateRef: TemplateRef<Object>) {
        authorizationService.getCurrentUser().then(user => {
            if (user.anonymous) {
                viewContainer.createEmbeddedView(templateRef);
            }
        });
    }
}
