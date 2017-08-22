import { Component, OnInit, OnDestroy } from '@angular/core';
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
    private viewPreferences: ViewPreferences;

    constructor(
        private activatedRoute: ActivatedRoute,
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

    getCommitByRevision(revisionId: string) {
        this.commitsService.getCommitByRevision(revisionId, true).subscribe(success => {
            this.commit =  success || new Commit();
            this.commitLoading = false;
        });
    }

    getTasks(offset: number, revision: string, excludeActive?: boolean) {
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
            this.onScrollLoading = false;
        });
    }

    openServiceTemplatePanel(commit: Commit) {
        this.launchPanelService.openPanel(commit);
    }

    onScroll() {
        if (this.canScroll) {
            this.onScrollLoading = true;
            this.getTasks(this.tasks.length, this.revisionId, true);
        }
    }
}
