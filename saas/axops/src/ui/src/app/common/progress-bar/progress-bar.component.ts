import { Component, Input } from '@angular/core';

import { TaskStatus } from '../../model';

@Component({
    selector: 'ax-progress-bar',
    templateUrl: './progress-bar.html',
    styles: [ require('./progress-bar.scss') ],
})
export class ProgressBarComponent {
    @Input()
    public status: TaskStatus = TaskStatus.Running;

    @Input()
    public progress: number = 0;

    get barClasses() {
        let barClasses = {};
        switch (this.status) {
            case TaskStatus.Failed:
                barClasses['progress-bar--failed'] = true;
                break;
            case TaskStatus.Running:
            case TaskStatus.Canceling:
                barClasses['progress-bar--running'] = true;
                break;
            case TaskStatus.Success:
                barClasses['progress-bar--success'] = true;
                break;
            case TaskStatus.Waiting:
                barClasses['progress-bar--waiting'] = true;
                break;
            default:
                barClasses['progress-bar--waiting'] = true;
                break;
        }
        if (this.progress >= 100) {
            barClasses['progress-bar--full'] = true;
        }
        return barClasses;
    }
}
