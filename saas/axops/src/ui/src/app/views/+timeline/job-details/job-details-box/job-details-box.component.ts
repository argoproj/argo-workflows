import { Component, Input, OnChanges } from '@angular/core';

import { Task, TaskStatus } from '../../../../model';
import { ViewUtils } from '../../../../common';

@Component({
    selector: 'ax-job-details-box',
    templateUrl: './job-details-box.html',
    styles: [ require('./job-details-box.scss') ],
})
export class JobDetailsBoxComponent implements OnChanges {
    @Input()
    task: Task;

    public artifactTags: string[];
    public isVisibleCancelButton: boolean = false;
    public labels: string[];

    ngOnChanges() {
        this.isVisibleCancelButton = [TaskStatus.Cancelled, TaskStatus.Failed, TaskStatus.Success].indexOf(this.task.status) === -1;
        this.artifactTags = this.task.artifact_tags ? JSON.parse(this.task.artifact_tags) : [];
        this.labels = this.task.labels ? ViewUtils.mapLabelsToList(this.task.labels) : [];
    }
}
