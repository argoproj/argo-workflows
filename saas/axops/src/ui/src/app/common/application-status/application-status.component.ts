import { Component, Input } from '@angular/core';

@Component({
    selector: 'ax-application-status',
    templateUrl: './application-status.html',
    styles: [ require('./application-status.scss') ]
})
export class ApplicationStatusComponent {
    @Input()
    status: string;
}
