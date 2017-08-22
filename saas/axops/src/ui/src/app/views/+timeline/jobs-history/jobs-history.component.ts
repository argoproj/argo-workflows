import { Component, OnDestroy, Input } from '@angular/core';
import { Subscription } from 'rxjs';

import { Task, Template, TaskStatus } from '../../../model';
import { TaskService } from '../../../services';

@Component({
    selector: 'ax-jobs-history',
    templateUrl: './jobs-history.html',
    styles: [ require('./jobs-history.scss') ],
})

export class JobsHistoryComponent implements OnDestroy {
    @Input()
    template: Template;

    public canLoadMore: boolean = false;
    public tasks: Task[] = [];
    public dataLoaded: boolean = false;
    public jobsCounterByType: { waiting: number, scheduled: number };

    private offset: number = 0;
    private readonly limit: number = 20;
    private subscription: Subscription;

    constructor(private taskService: TaskService) {
    }

    ngOnDestroy() {
        if (this.subscription) {
            this.subscription.unsubscribe();
        }
    }

    public onLoadMore() {
        if (this.canLoadMore) {
            this.canLoadMore = false;
            this.getJobsHistory();
        }
    }

    public loadJobsHistory() {
        this.jobsCounterByType = { waiting: 0, scheduled: 0 };
        this.getJobsHistory();
    }

    public clearJobsHistory() {
        this.tasks = [];
        this.offset = 0;
    }

    private getJobsHistory() {
        if (!this.template) {
            return false;
        }

        this.dataLoaded = false;
        let params = {
            templateIds: this.template ? this.template.id : null,
            limit: this.limit,
            offset: this.offset,
            fields: ['id', 'name', 'description', 'status', 'template', 'username', 'cost']
        };

        if (this.offset > 0) {
            params['isActive'] = false;
        }

        this.subscription = this.taskService.getTasks(params, true)
            .subscribe(
                success => {
                    if (this.offset === 0) {
                        success.data.forEach(task => {
                            this.jobsCounterByType.scheduled += (task.status === TaskStatus.Running || task.status === TaskStatus.Canceling) ? 1 : 0;
                            this.jobsCounterByType.waiting += (task.status === TaskStatus.Waiting || task.status === TaskStatus.Init) ? 1 : 0;
                        });
                    }
                    this.offset += this.limit;
                    this.canLoadMore = success.data.length >= this.limit;
                    this.tasks = this.tasks.concat(success.data);
                    this.dataLoaded = true;
                }
            );
    }
}
