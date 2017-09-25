import * as moment from 'moment';
import { Component, OnDestroy, ViewChildren, QueryList, Input, OnChanges, SimpleChanges, NgZone } from '@angular/core';
import { Subscription } from 'rxjs';

import { BranchTasks, TaskStatus, Branch, Task } from '../../../model';
import { TaskService } from '../../../services';
import { BranchInfo, JobFilter, NowLine } from '../branches.view-models';
import { JobsTimelineComponent } from '../jobs-timeline/jobs-timeline.component';
import { DateRange } from 'argo-ui-lib/src/components';
import { ViewUtils } from '../../../common';
import { SortOperations } from '../../../common/sortOperations/sortOperations';
import { RepoNamePipe } from '../../../pipes/repoName.pipe';

@Component({
    selector: 'ax-branches-overview',
    templateUrl: './branches-overview.html',
    styles: [ require('./branches-overview.scss') ],
})
export class BranchesOverviewComponent implements OnChanges, OnDestroy {

    @Input()
    public selectedRepo: string = null;
    @Input()
    public selectedBranch: string = null;
    @Input()
    public dateRange: DateRange;
    @Input()
    public jobFilter: JobFilter = new JobFilter();
    @Input()
    public searchString: string;
    @Input()
    public branchesContext: Branch[];

    public branches: BranchInfo[];
    public nowLine: NowLine = new NowLine();

    @ViewChildren(JobsTimelineComponent)
    public jobsTimelines: QueryList<JobsTimelineComponent>;

    private preferencesSubscription: Subscription;
    private eventsSubscription: Subscription;
    private loadTasksSubscription: Subscription;

    constructor(private taskService: TaskService, private zone: NgZone) {
    }

    public ngOnChanges(changes: SimpleChanges) {
        if (changes['dateRange'] && !DateRange.equals(changes['dateRange'].currentValue, changes['dateRange'].previousValue)
            || changes['selectedRepo']
            || changes['selectedBranch']
            || changes['branchesContext']) {
            this.loadTasks();
        }
    }

    public ngOnDestroy() {
        if (this.preferencesSubscription != null) {
            this.preferencesSubscription.unsubscribe();
            this.preferencesSubscription = null;
        }
        if (this.eventsSubscription) {
            this.eventsSubscription.unsubscribe();
            this.eventsSubscription = null;
        }
        this.loadTasksUnsubscribe();
    }

    public getBranchRoute(branch: Branch) {
        return ['/app/timeline', ViewUtils.sanitizeRouteParams(
            { view: 'job', repo: branch.repo, branch: branch.name }, this.dateRange.toRouteParams(), this.jobFilter)];
    }

    private loadTasks() {
        this.loadTasksUnsubscribe();

        this.branches = null;
        if (!this.branchesContext || this.branchesContext.length > 0) {
            let params = {
                startTime: this.dateRange.startDate,
                endTime: this.dateRange.endDate,
                repo: this.selectedRepo,
                branch: this.selectedBranch,
                branches: this.branchesContext,
            };
            // Don't load active tasks if date range does not include today's date
            if (!this.dateRange.containsToday) {
                params['isActive'] = false;
            }

            this.loadTasksSubscription = this.taskService.getTasksByBranches(params, true).subscribe((branchTasks: BranchTasks[]) => {
                this.branches = SortOperations.sortBy(branchTasks
                    .map(item => this.getBranchInfo(item)), 'shortcutRepoBranch', true);
                if (this.branchesContext) {
                    let repoBranchToInfo = new Map<string, BranchInfo>();
                    this.branches.forEach(info => repoBranchToInfo.set(`${info.repo}_${info.name}`, info));
                    this.branchesContext.forEach(branch => {
                        if (!repoBranchToInfo.has(`${branch.repo}_${branch.name}`)) {
                            this.branches.push(this.getBranchInfo({
                                branch: branch.name,
                                repo: branch.repo,
                                tasks: [],
                            }));
                        }
                    });
                }
                this.calculateNowLine();
                this.subscribeToEvents();
            });
        } else {
            this.branches = [];
            this.subscribeToEvents();
        }
    }

    private calculateNowLine(): void {
        let now = moment();
        if (now.isBetween(this.dateRange.startDate, this.dateRange.endDate, 'day', '[]')) {
            let duration = this.dateRange.endDate.diff(this.dateRange.startDate);
            this.nowLine = {
                left: `${now.diff(this.dateRange.startDate) / duration * 100}%`,
                now: now.valueOf(),
                inRange: true
            };
        } else {
            this.nowLine.inRange = false;
        }
    }

    private getBranchInfo(input: BranchTasks): BranchInfo {
        return {
            name: input.branch,
            repo: input.repo,
            shortcutRepoBranch: `${new RepoNamePipe().transform(input.repo)}/${input.branch}`,
            failedJobsCount: input.tasks.filter(task => task.status === TaskStatus.Failed).length,
            scheduledJobsCount: input.tasks.filter(task => task.status === TaskStatus.Init || task.status === TaskStatus.Waiting).length,
            successfulJobsCount: input.tasks.filter(task => task.status === TaskStatus.Success).length,
            runningJobsCount: input.tasks.filter(task => task.status === TaskStatus.Running || task.status === TaskStatus.Canceling).length,
            canceledJobsCount: input.tasks.filter(task => task.status === TaskStatus.Cancelled).length,
            timelineInput: {
                tasks: input.tasks,
                startTime: this.dateRange.startDate,
                endTime: this.dateRange.endDate
            }
        };
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
            if (eventInfo.branch && eventInfo.repo && eventInfo.id === eventInfo.task_id) {
                let info = this.branches.find(branchInfo => branchInfo.name === eventInfo.branch && branchInfo.repo === eventInfo.repo);
                if (info) {
                    let tasks = info.timelineInput.tasks;
                    let task = tasks.find(item => item.id === eventInfo.id);
                    if (!task) {
                        task = Object.assign(new Task(), { id: eventInfo.id });
                        tasks.unshift(task);
                    }
                    task.status = eventInfo.status;

                    this.zone.run(() => {
                        Object.assign(info, this.getBranchInfo({ branch: info.name, repo: info.repo, tasks: tasks }));
                    });
                } else {
                    let tasks = [Object.assign(new Task(), { id: eventInfo.id, status: eventInfo.status })];
                    let netInfo = this.getBranchInfo({ branch: eventInfo.branch, repo: eventInfo.repo, tasks: tasks });
                    this.zone.run(() => {
                        this.branches.unshift(netInfo);
                        SortOperations.sortBy(this.branches, 'shortcutRepoBranch', true);
                    });
                }
            }
        });
    }

    private loadTasksUnsubscribe() {
        if (this.loadTasksSubscription) {
            this.loadTasksSubscription.unsubscribe();
            this.loadTasksSubscription = null;
        }
    }
}
