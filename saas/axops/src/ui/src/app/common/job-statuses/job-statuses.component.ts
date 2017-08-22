import { Component, Input } from '@angular/core';

@Component({
    selector: 'ax-job-statuses',
    templateUrl: './job-statuses.component.html',
    styles: [ require('./job-statuses.scss') ],
})
export class JobStatusesComponent {
    @Input()
    public successful;

    @Input()
    public failed;

    @Input()
    public inProgress;

    @Input()
    public queued;
}
