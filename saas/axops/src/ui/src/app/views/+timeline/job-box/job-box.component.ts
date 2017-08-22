import { Component, Input } from '@angular/core';

import { Task } from '../../../model';

@Component({
    selector: 'ax-job-box',
    templateUrl: './job-box.html',
    styles: [ require('./job-box.scss') ],
})
export class JobBoxComponent {
    @Input()
    task: Task;
    @Input()
    contextMenu: boolean = true;

    public get commitHistoryLink(): string {
        return this.task ? `/app/commits/history/${encodeURIComponent(this.task.commit.repo)}/${this.task.commit.revision}` : '';
    }
}
