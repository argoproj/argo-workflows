import * as moment from 'moment';
import { Component, Input } from '@angular/core';

import { Task, TaskStatus } from '../../../model';
import { JobsService } from '../jobs.service';
import { JobsTimelineInput, JobFilter, NowLine } from '../branches.view-models';


@Component({
    selector: 'ax-jobs-timeline',
    templateUrl: './jobs-timeline.html',
    styles: [ require('./jobs-timeline.scss') ],
})
export class JobsTimelineComponent {
    @Input()
    enableClickNavigation: boolean = false;
    @Input()
    nowLine: NowLine = new NowLine();
    @Input()
    runningJobsCount: number;
    @Input()
    scheduledJobsCount: number;
    @Input()
    jobFilter: JobFilter = new JobFilter();

    private jobs: {left: string, task: Task}[] = [];

    static getJobType(taskStatus: TaskStatus): string {
        let type: string;
        switch (taskStatus) {
            case TaskStatus.Failed:
                type = 'failed';
                break;
            case TaskStatus.Cancelled:
                type = 'failed';
                break;
            case TaskStatus.Init:
                type = 'deployed';
                break;
            case  TaskStatus.Waiting:
                type = 'deployed';
                break;
            case  TaskStatus.Success:
                type = 'succeeded';
                break;
            case TaskStatus.Running:
                type = 'running';
                break;
            default:
                break;
        }
        return type;
    }

    constructor(private jobsService: JobsService) {
    }

    onTaskClicked(task: Task) {
        if (this.enableClickNavigation) {
            this.jobsService.showJob.emit(task);
        }
    }

    getTaskClasses(task: Task): any {
        return {
            'jobs-timeline__failed': task.status === TaskStatus.Failed || task.status === TaskStatus.Cancelled,
            'jobs-timeline__deployed': task.status === TaskStatus.Init || task.status === TaskStatus.Waiting,
            'jobs-timeline__succeeded': task.status === TaskStatus.Success,
            'jobs-timeline__clickable': this.enableClickNavigation,
            'jobs-timeline__filtered': this.jobFilter[JobsTimelineComponent.getJobType(task.status)]
        };
    }

    @Input()
    set input(input: JobsTimelineInput) {
        let duration = input.endTime.diff(input.startTime);
        this.jobs = input.tasks
            .sort((first, second) => first.create_time - second.create_time)
            .filter((task, index, tasks) => {
                return task.status !== TaskStatus.Running &&
                       task.status !== TaskStatus.Init &&
                    (
                        index === 0 ||
                        (task.status !== tasks[index - 1].status) ||
                        (
                            (task.status === tasks[index - 1].status) &&
                            (task.create_time !== tasks[index - 1].create_time)
                        )
                    );
            })
            .map(task => {
                let left = 0;
                let startTime;
                if (task.status === TaskStatus.Failed) {
                    startTime = moment.unix(task.launch_time + task.run_time);
                } else if (
                    (task.status !== TaskStatus.Failed && task.status !== TaskStatus.Running) &&
                    moment.unix(task.create_time) >= input.startTime &&
                    moment.unix(task.create_time) <= input.endTime
                ) {
                    startTime = moment.unix(task.create_time);
                }
                if (startTime) {
                    left = startTime.diff(input.startTime) / duration * 100;
                }
                return {
                    task: task,
                    left: `${left}%`
                };
            });
    }
}
