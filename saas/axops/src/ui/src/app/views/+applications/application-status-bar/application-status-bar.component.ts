import { Component, Input } from '@angular/core';

import { Application } from '../../../model';

@Component({
    selector: 'ax-application-status-bar',
    templateUrl: './application-status-bar.html',
    styles: [ require('./application-status-bar.scss') ]
})
export class ApplicationStatusBarComponent {

    @Input()
    public application: Application;

}
