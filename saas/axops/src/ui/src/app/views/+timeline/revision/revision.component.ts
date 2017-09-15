import { Component, OnInit, OnDestroy, NgZone } from '@angular/core';
import { ActivatedRoute } from '@angular/router';
import { Subscription } from 'rxjs';

import { LayoutSettings, HasLayoutSettings } from '../../layout/layout.component';
import { CommitsService, TaskService, ViewPreferencesService } from '../../../services';
import { Task, TaskFieldNames, Commit, ViewPreferences } from '../../../model';
import { LaunchPanelService, ViewUtils } from '../../../common';

@Component({
    selector: 'ax-revision',
    templateUrl: './revision.html',
    styles: [ require('./revision.scss') ],
})
export class RevisionComponent implements OnInit, OnDestroy, LayoutSettings, HasLayoutSettings {
    public onScrollLoading: boolean = false;
    public taskLoading: boolean = false;
    public commitLoading: boolean = false;

    public searchString: string = '';

    private commit: Commit = new Commit();
    private tasks: Task[] = [];
    private revisionId: string = '';
    private canScroll: boolean = false;
    private subscriptions: Subscription[] = [];
    private idToTask: Map<string, Task> = new Map<string, Task>();
    private viewPreferences: ViewPreferences;
    private eventsSubscription: Subscription;

    constructor(
        private activatedRoute: ActivatedRoute,
        private zone: NgZone,
        private commitsService: CommitsService,
        private taskService: TaskService,
        private launchPanelService: LaunchPanelService,
        private viewPreferencesService: ViewPreferencesService) {
    }

    public ngOnInit() {
        this.subscriptions.push(this.viewPreferencesService.getViewPreferencesObservable().subscribe(viewPreferences => {
            this.viewPreferences = viewPreferences;
        }));

        this.activatedRoute.params.subscribe(params => {
            this.taskLoading = true;
            this.commitLoading = true;
            this.revisionId = params['revisionId'];
            this.getCommitByRevision(this.revisionId);
            this.getTasks(0, this.revisionId);
        });
    }

    public ngOnDestroy() {
        this.eventSubscriptionUnsubscribe();
        this.subscriptions.forEach(item => item.unsubscribe());
        this.subscriptions = [];
    }

    get layoutSettings(): LayoutSettings {
        return this;
    }

    public branchNavPanelUrl = '/app/timeline';

    get pageTitle(): string {
        return 'Timeline';
    }

    get breadcrumb(): { title: string, routerLink?: any[] }[] {
        return this.commit ? ViewUtils.getBranchBreadcrumb(this.commit.repo, this.commit.branch, '/app/timeline', this.viewPreferences, this.commit.revision) : null;
    }

    public openServiceTemplatePanel(commit: Commit) {
        this.launchPanelService.openPanel(commit);
    }

    public onScroll() {
        if (this.canScroll) {
            this.onScrollLoading = true;
            this.getTasks(this.tasks.length, this.revisionId, true);
        }
    }

    private getTasks(offset: number, revision: string, excludeActive?: boolean) {
        const pageSize = 20;
        this.canScroll = false;
        let params: any = {
            revision: revision,
            limit: pageSize,
            offset: offset,
            fields: [
                TaskFieldNames.name,
                TaskFieldNames.status,
                TaskFieldNames.commit,
                TaskFieldNames.username,
                TaskFieldNames.failurePath
            ],
        };
        if (excludeActive) {
            params.isActive = false;
        }
        this.taskService.getTasks(params).subscribe(success => {
            this.tasks = this.tasks.concat(success.data || []);
            this.canScroll = (success.data || []).length >= pageSize;
            this.taskLoading = false;
            this.idToTask = new Map<string, Task>();
            (this.tasks || []).forEach(task => this.idToTask.set(task.id, task));
            this.onScrollLoading = false;
        });
    }

    private subscribeToEvents() {
        this.eventSubscriptionUnsubscribe();

        this.eventsSubscription = this.taskService.getTasksEvents(this.commit.repo, this.commit.branch).subscribe(eventInfo => {
            // handle only root workflow events
            if (this.tasks && eventInfo.repo && eventInfo.id === eventInfo.task_id) {

                let tasks = this.tasks;
                let task = this.idToTask.get(eventInfo.id);
                if (task) {
                    this.zone.run(() => {
                        task.status = eventInfo.status;
                    });
                } else {
                    let newTask = Object.assign(new Task(), { id: eventInfo.id, status: eventInfo.status, template: null });
                    this.idToTask.set(newTask.id, newTask);
                    this.zone.run(() => {
                        this.taskService.getTask(eventInfo.id, true, true).subscribe(newTaskData => tasks.unshift(Object.assign(newTask, newTaskData)));
                    });
                }
            }
        });
    }

    private eventSubscriptionUnsubscribe() {
        if (this.eventsSubscription) {
            this.eventsSubscription.unsubscribe();
            this.eventsSubscription = null;
        }
    }

    private getCommitByRevision(revisionId: string) {
        this.commitsService.getCommitByRevision(revisionId, true).subscribe(success => {
            this.commit =  success || new Commit();
            this.commitLoading = false;

            this.subscribeToEvents();
        });
    }
}
