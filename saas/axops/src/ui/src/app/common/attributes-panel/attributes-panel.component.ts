import { Component, Input  } from '@angular/core';

@Component({
    selector: 'ax-attributes-panel',
    templateUrl: './attributes-panel.html',
})
export class AttributesPanelComponent {
    @Input()
    public attributes: any;
    @Input()
    public title = 'Attributes';
    @Input()
    public noAttributesMessage = 'There are no attributes';

    public get hasNoAttributes(): boolean {
        return Object.keys(this.attributes || {}).length === 0;
    }
}
