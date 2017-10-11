import { Component, OnChanges, Input, OnDestroy, NgZone } from '@angular/core';
import { Subscription, Observable } from 'rxjs';

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
    public dataLoaded: boolean = false;
    public showLoaderMockup: boolean = false;
    public tasks: Task[];
    private tasksBuffer: Task[] = [];
    private idToTask: Map<string, Task> = new Map<string, Task>();
    private canScroll: boolean = false;
    private offset: number = 0;
    private eventsSubscription: Subscription;
    private getTasksSubscription: Subscription;

    constructor(private taskService: TaskService, private zone: NgZone) {
    }

    public ngOnChanges() {
        this.getTasksUnsubscribe();

        this.tasks = null;
        this.idToTask = new Map<string, Task>();
        this.getTasksSubscription = this.loadNextTasksPage(true, true);

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
            this.loadNextTasksPage(false, true);
        }
    }

    private loadNextTasksPage(isFirstLoad: boolean, hideLoader?: boolean): Subscription {
        if (isFirstLoad) {
            this.tasksBuffer = [];
            this.tasks = [];
            this.offset = 0;
            this.showLoaderMockup = true;
        }
        let tasksObservable: Observable<Task[]>;
        // Client side pagination for active tasks. This is required since backend does not support active tasks pagination.
        if (this.tasksBuffer.length > 0) {
            let res = this.tasksBuffer.slice(0, this.bufferSize);
            this.tasksBuffer = this.tasksBuffer.slice(this.bufferSize);
            tasksObservable = Observable.from([res]);
            this.canScroll = true;
        } else {
            let params = {
                startTime: null,
                endTime: null,
                limit: this.bufferSize,
                offset: this.offset,
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

            if (this.jobFilter && !this.jobFilter.allSelected) {
                params.status = this.getFilteredStatuses();
            }

            tasksObservable = this.taskService.getTasks(params, hideLoader).map(res => {
                let tasks = res.data.slice(0, this.bufferSize);
                this.tasksBuffer = res.data.slice(this.bufferSize);
                this.canScroll = tasks.length === this.bufferSize;
                this.offset = this.offset + tasks.length;
                return tasks;
            });
        }

        return tasksObservable.subscribe(tasks => {
            this.tasks = this.tasks.concat(tasks);
            this.idToTask = new Map<string, Task>();
            (this.tasks || []).forEach(task => this.idToTask.set(task.id, task));
            this.dataLoaded = true;
            this.showLoaderMockup = false;
        });
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
            if (this.tasks && eventInfo.id === eventInfo.task_id) {
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
