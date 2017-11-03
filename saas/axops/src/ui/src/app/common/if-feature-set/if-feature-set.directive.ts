import { Directive, ViewContainerRef, TemplateRef, Input } from '@angular/core';

import { FeaturesSetsService } from '../../services';

@Directive({
    selector: '[axIfFeatureSet]',
})
export class IfFeatureSetDirective {
    constructor(private authorizationService: FeaturesSetsService, private viewContainer: ViewContainerRef, private templateRef: TemplateRef<Object>) {
    }

    @Input() set axIfFeatureSet(value: string | string[]) {
        this.authorizationService.getFeaturesSet().then(featureSet => {
            let sets = typeof value === 'string' ? [value] : <string[]> value;
            if (!!sets.find(set => featureSet === set)) {
                this.viewContainer.createEmbeddedView(this.templateRef);
            } else {
                this.viewContainer.clear();
            }
        });
    }
}
