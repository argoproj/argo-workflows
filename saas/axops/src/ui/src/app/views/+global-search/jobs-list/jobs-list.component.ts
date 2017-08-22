import { Component, Input, OnChanges, OnDestroy } from '@angular/core';
import { Subscription } from 'rxjs';

import { NotificationsService } from 'argo-ui-lib/src/components';
import { TaskFieldNames, Task, TaskStatus } from '../../../model';
import { TaskService, GlobalSearchService, ModalService } from '../../../services';
import { StatusToNumberPipe } from '../../../pipes/statusToNumber.pipe';
import { Pagination, JobsFilters } from '../../../common';

import { BulkUpdater } from '../bulk-updater';

@Component({
    selector: 'ax-jobs-list',
    templateUrl: './jobs-list.html',
    styles: [require('./jobs-list.scss')],
})
export class JobsListComponent implements OnChanges, OnDestroy {

    @Input()
    public filters: JobsFilters;

    @Input()
    public searchString: string;

    public bulkUpdater: BulkUpdater<Task>;
    public tasks: Task[] = [];

    public limit: number = 10;
    public params: JobsFilters;
    public dataLoaded: boolean = false;
    public pagination: Pagination = {
        limit: this.limit,
        offset: 0,
        listLength: this.tasks.length
    };
    private activeTasks: Task[] = [];
    private subscriptions: Subscription[] = [];
    private getTasksSubscrioption: Subscription;

    constructor(private taskService: TaskService,
                private globalSearchService: GlobalSearchService,
                modalService: ModalService,
                notificationsService: NotificationsService) {
        this.bulkUpdater = new BulkUpdater<Task>(modalService, notificationsService)
            .addAction('cancel', {
                title: 'Cancel Jobs',
                confirmation: count => `Are you sure you want to cancel ${count} jobs?`,
                execute: task => this.taskService.cancelTask(task.id).toPromise(),
                isApplicable: task => this.isActiveTask(task) && task.status !== TaskStatus.Canceling,
                warningMessage: tasks => `The ${tasks.length} selected job${tasks.length > 1 ? 's' : ''} are not active and cannot be cancelled.`,
                postMessage: (successfulCount, failedCount) => `${successfulCount} jobs had been successfully canceled.`
            }).addAction('resubmit', {
                title: 'Resubmit Jobs',
                confirmation: count => `Are you sure you want to resubmit ${count} Job${count > 1 ? 's' : ''}?`,
                execute: task => this.resubmitTask(task.id),
                isApplicable: task => true,
                warningMessage: tasks => null,
                postMessage: (successfulCount, failedCount) => `${successfulCount} jobs had been successfully resubmitted.`
            });
            this.bulkUpdater.actionExecuted.subscribe(action => this.updateTasks(this.params, this.pagination, true));
    }

    private resubmitTask(taskId: string): Promise<any> {
        return this.taskService.getTask(taskId).toPromise().then(task => {
            return this.taskService.launchTask({
                arguments: task.arguments,
                template_id: task.template_id,
            }).toPromise();
        });
    }

    public ngOnChanges() {
        // need to map readable statuses string representation to numbers
        this.params = {
            statuses: this.filters.statuses.map(status => new StatusToNumberPipe().transform(status).toString()),
            authors: this.filters.authors,
            artifact_tags: this.filters.artifact_tags,
            branch: this.filters.branch,
            repo: this.filters.repo
        };
        // restart pagination if changed search parameters
        this.pagination = {limit: this.limit, offset: 0, listLength: this.tasks.length};
        this.updateTasks(this.params, this.pagination, true);
    }

    public ngOnDestroy() {
        this.subscriptions.forEach(subscription => subscription.unsubscribe());
        this.subscriptions = [];
        this.unsubscribeGetTaskSubscription();
    }

    public onPaginationChange(pagination: Pagination) {
        this.limit = pagination.limit;
        this.updateTasks(this.params, {offset: pagination.offset, limit: pagination.limit}, true);

        // unselect all
        this.bulkUpdater.clearSelection();
    }

    public navigateToDetails(id: string): void {
        this.globalSearchService.navigate(['/app/timeline/jobs/', id]);
    }

    private isActiveTask(task: Task) {
        return task.status !== TaskStatus.Cancelled && task.status !== TaskStatus.Success && task.status !== TaskStatus.Failed ? 1 : 0;
    }

    private updateTasks(params: JobsFilters, pagination: Pagination, hideLoader?: boolean) {
        this.unsubscribeGetTaskSubscription();

        this.dataLoaded = false;
        // Remove previously loaded number of active tasks from offset, since pagination is not applicable to active tasks
        let offset = Math.max(0, pagination.offset - this.activeTasks.length);
        this.getTasksSubscrioption = this.getTasks(params, this.limit + 1, offset)
        // Sort tasks by status to make sure that active tasks come first
            .map(result => result.data.sort((first, second) => this.isActiveTask(second) - this.isActiveTask(first))).subscribe(tasks => {

                this.activeTasks = tasks.filter(task => this.isActiveTask(task));

                this.dataLoaded = true;

                // Remove active tasks which are not supposed to be shown for current page
                let prevPagesTaskCount = Math.min(pagination.offset, this.activeTasks.length);
                this.tasks = tasks.slice(prevPagesTaskCount, prevPagesTaskCount + pagination.limit).map(task => {
                    task.artifact_tags = task.artifact_tags.length ? JSON.parse(task.artifact_tags) : [];
                    return task;
                });
                this.bulkUpdater.items = this.tasks;

                this.pagination = {
                    offset: pagination.offset,
                    limit: this.limit,
                    listLength: this.tasks.length,
                    hasMore: tasks.filter(task => !this.isActiveTask(task)).length > this.limit
                };
            }, error => {
                this.dataLoaded = true;
                this.tasks = [];
                this.bulkUpdater.items = this.tasks;
            });
    }

    private unsubscribeGetTaskSubscription() {
        if (this.getTasksSubscrioption) {
            this.getTasksSubscrioption.unsubscribe();
            this.getTasksSubscrioption = null;
        }
    }

    private getTasks(params: JobsFilters, limit: number, offset: number) {
        let parameters = {
            status: null,
            tags: null,
            limit: null,
            offset: null,
            repo: null,
            branches: null,
            username: null,
            fields: [
                TaskFieldNames.name,
                TaskFieldNames.status,
                TaskFieldNames.status_string,
                TaskFieldNames.username,
                TaskFieldNames.commit,
                TaskFieldNames.repo,
                TaskFieldNames.branch,
            ],
            searchFields: [
                TaskFieldNames.name,
                TaskFieldNames.description,
                TaskFieldNames.status_string,
                TaskFieldNames.username,
                TaskFieldNames.repo,
                TaskFieldNames.branch,
            ],
            search: this.searchString
        };

        parameters.offset = offset;
        parameters.limit = limit;

        if (params.statuses && params.statuses.length) {
            parameters.status = params.statuses;
        }

        if (params.authors && params.authors.length) {
            parameters.username = params.authors;
        }

        if (params.artifact_tags && params.artifact_tags.length) {
            parameters.tags = params.artifact_tags;
        }

        if (params.repo && params.repo.length) {
            if (params.repo.length === 1 && params.branch.length === 0) {
                parameters.repo = params.repo;
            } else {
                parameters.branches = params.repo.map(i => {
                    return {repo: i, name: ''};
                });
            }
        }

        if (params.branch && params.branch.length) {
            let branches = params.branch.map(i => {
                return {repo: i.split(' ')[0], name: i.split(' ')[1]};
            });

            parameters.branches = parameters.branches || [];
            parameters.branches = parameters.branches.concat(branches || []);
        }

        return this.taskService.getTasks(parameters, true);
    }
}
