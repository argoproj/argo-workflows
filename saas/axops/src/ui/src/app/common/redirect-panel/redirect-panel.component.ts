import { Component, Input } from '@angular/core';

@Component({
    selector: 'ax-redirect-panel',
    templateUrl: './redirect-panel.html',
    styles: [ require('./redirect-panel.scss') ]
})
export class RedirectPanelComponent {

    @Input()
    public title: string;

    @Input()
    public description: string;

    @Input()
    public route: string[];
}
