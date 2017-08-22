import { Component, OnChanges, Input, OnDestroy, NgZone } from '@angular/core';
import { Subscription } from 'rxjs';

import { DateRange } from 'argo-ui-lib/src/components';
import { TaskService } from '../../../services';
import { Task, TaskStatus, TaskFieldNames, Branch } from '../../../model';
import { JobFilter } from '../branches.view-models';

@Component({
    selector: 'ax-jobs-overview',
    templateUrl: './jobs-overview.html',
    styles: [ require('./jobs-overview.scss') ],
})
export class JobsOverviewComponent implements OnChanges, OnDestroy {
    protected readonly bufferSize: number = 20;

    @Input()
    public selectedRepo: string = null;
    @Input()
    public selectedBranch: string = null;
    @Input()
    public selectedUsername: string = null;
    @Input()
    public searchString: string = null;
    @Input()
    public dateRange: DateRange;
    @Input()
    public jobFilter: JobFilter = new JobFilter();
    @Input()
    public branchesContext: Branch[];

    public hideLabels: boolean = true;
    private tasks: Task[];
    private idToTask: Map<string, Task> = new Map<string, Task>();
    private canScroll: boolean = false;
    private offset: number = 0;
    private dataLoaded: boolean = false;
    private eventsSubscription: Subscription;
    private getTasksSubscription: Subscription;

    constructor(private taskService: TaskService, private zone: NgZone) {
    }

    public ngOnChanges() {
        this.getTasksUnsubscribe();

        this.tasks = null;
        this.indexTasks();
        this.getTasksSubscription = this.getTasks(0, this.bufferSize, true, true).subscribe(res => {
            this.tasks = res.data;
            this.indexTasks();
            // Force loading second page of data after first page since pagination is not applied to active/schedule tasks
            this.canScroll = res.data.length === this.bufferSize || this.dateRange.containsToday;
            this.offset = this.offset + res.data.length;
            this.dataLoaded = true;
        });

        this.subscribeToEvents();
    }

    public ngOnDestroy() {
        if (this.eventsSubscription) {
            this.eventsSubscription.unsubscribe();
            this.eventsSubscription = null;
        }

        this.getTasksUnsubscribe();
    }


    public onScroll() {
        if (this.canScroll) {
            this.canScroll = false;
            this.dataLoaded = false;
            this.getTasks(this.offset, this.bufferSize, false, true).subscribe(res => {
                this.dataLoaded = true;
                this.tasks = this.tasks.concat(res.data);
                this.indexTasks();
                this.canScroll = res.data.length === this.bufferSize;
                this.offset = this.offset + res.data.length;
            });
        }
    }

    private getTasks(skip: number, limit: number, isFirstLoad: boolean, hideLoader?: boolean) {
        let params = {
            startTime: null,
            endTime: null,
            limit: null,
            offset: null,
            repo: this.selectedRepo,
            branch: this.selectedBranch,
            branches: this.selectedRepo || this.selectedBranch ? null : this.branchesContext,
            username: [this.selectedUsername],
            fields: [TaskFieldNames.name,
                TaskFieldNames.status,
                TaskFieldNames.commit,
                TaskFieldNames.failurePath,
                TaskFieldNames.labels,
                TaskFieldNames.username,
                TaskFieldNames.templateId,
                TaskFieldNames.parameters,
                TaskFieldNames.jira_issues,
                TaskFieldNames.policy_id,
            ],
            search: this.searchString,
            status: [],
        };
        // Don't load active tasks if date range does not include today's date and does not load active tasks during pagination
        if ( !isFirstLoad || !this.dateRange.containsToday) {
            params['isActive'] = false;
        }

        if (!this.dateRange.isAllDates) {
            params.startTime = this.dateRange.startDate;
            params.endTime = this.dateRange.endDate;
        }

        if (skip) {
            params.offset = skip;
        }

        if (limit) {
            params.limit = limit;
        }

        if (this.jobFilter && !this.jobFilter.allSelected) {
            params.status = this.getFilteredStatuses();
        }

        params.search = this.searchString;

        return this.taskService.getTasks(params, hideLoader);
    }


    private indexTasks() {
        this.idToTask = new Map<string, Task>();
        (this.tasks || []).forEach(task => this.idToTask.set(task.id, task));
    }

    private subscribeToEvents() {
        if (this.eventsSubscription) {
            this.eventsSubscription.unsubscribe();
            this.eventsSubscription = null;
        }
        if (!this.dateRange.containsToday) {
            return;
        }
        this.eventsSubscription = this.taskService.getTasksEvents(this.selectedRepo, this.selectedBranch).subscribe(eventInfo => {
            // handle only root workflow events
            if (this.tasks && eventInfo.repo && eventInfo.id === eventInfo.task_id) {
                let tasks = this.tasks;
                let task = this.idToTask.get(eventInfo.id);
                if (task) {
                    this.zone.run(() => {
                        task.status = eventInfo.status;
                    });
                } else {
                    if (this.getFilteredStatuses().indexOf(eventInfo.status)) {
                        let newTask = Object.assign(new Task(), { id: eventInfo.id, status: eventInfo.status, template: null });
                        this.idToTask.set(newTask.id, newTask);
                        this.zone.run(() => {
                            this.taskService.getTask(eventInfo.id, true, true).subscribe(newTaskData => tasks.unshift(Object.assign(newTask, newTaskData)));
                        });
                    }
                }
            }
        });
    }

    private getJobStatus(type: string): number[] {
        let status: number[] = [];
        switch (type) {
            case 'failed':
                status = [ TaskStatus.Failed, TaskStatus.Cancelled ];
                break;
            case 'delayed':
                status = [ TaskStatus.Init, TaskStatus.Waiting ];
                break;
            case 'succeeded':
                status = [ TaskStatus.Success ];
                break;
            case 'running':
                status = [ TaskStatus.Running ];
                break;
            default:
                break;
        }
        return status;
    }

    private getFilteredStatuses() {
        let statuses = [];
        for (let key in (this.jobFilter || {})) {
            if (this.jobFilter[key] === true) {
                statuses = statuses.concat(this.getJobStatus(key));
            }
        }
        return statuses;
    }

    private getTasksUnsubscribe() {
        if (this.getTasksSubscription) {
            this.getTasksSubscription.unsubscribe();
            this.getTasksSubscription = null;
        }
    }
}
