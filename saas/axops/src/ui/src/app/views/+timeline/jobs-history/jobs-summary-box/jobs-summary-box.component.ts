import {Component, Input} from '@angular/core';


@Component({
    selector: 'ax-jobs-summary-box',
    templateUrl: './jobs-summary-box.html',
    styles: [ require('./jobs-summary-box.scss') ],
})

export class JobsSummaryBoxComponent {
    @Input()
    jobs_wait: number;
    @Input()
    jobs_run: number;
    @Input()
    jobs_fail: number;
    @Input()
    jobs_success: number;
}
