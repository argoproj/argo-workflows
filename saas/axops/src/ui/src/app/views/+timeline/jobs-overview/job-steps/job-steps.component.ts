import { Component, Input, Output, EventEmitter, OnInit } from '@angular/core';

import { Task, TaskStatus } from '../../../../model';
import { ToolService } from '../../../../services';

interface StepInfo {
    isSucceeded: boolean;
    isFailed: boolean;
    isRunning: boolean;
    name: string;
}

@Component({
    selector: 'ax-job-steps',
    templateUrl: './job-steps.html',
    styles: [ require('./job-steps.scss') ],
})
export class JobStepsComponent implements OnInit {
    @Input()
    public set setTask(value: Task) {
        this.task = value;
        if (value && value.commit && value.commit.committer) {
            this.commiterEmail = value.commit.committer.substring(value.commit.committer.lastIndexOf('<') + 1,
                value.commit.committer.trim().length - 1);
        }
        if (value && value.failure_path) {
            this.steps = value.failure_path.map(name => {
                let isSucceeded = value.status === TaskStatus.Success;
                let isFailed = value.status === TaskStatus.Failed || value.status === TaskStatus.Cancelled;
                return {
                    name,
                    isSucceeded: isSucceeded,
                    isFailed: isFailed,
                    isRunning: !isSucceeded && !isFailed
                };
            }).slice(-3);
        }
    }

    @Output()
    public onToggleIssues: EventEmitter<any> = new EventEmitter();

    public commiterEmail: string = '';
    public task: Task;
    public steps: StepInfo[] = [];
    public isOpen: boolean = false;
    public isJiraConfigured: boolean = false;

    constructor(private toolService: ToolService) {}

    public toggleIssuesPanel() {
        this.isOpen = !this.isOpen;
        this.onToggleIssues.emit(this.isOpen);
    }

    public ngOnInit() {
        this.toolService.isJiraConfigured().subscribe(isConfigured => this.isJiraConfigured = isConfigured);
    }
}
