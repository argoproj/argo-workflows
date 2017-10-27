import { Component, Input, OnInit, OnDestroy } from '@angular/core';

import { Subscription } from 'rxjs';

import { TimeFormatter } from '../../../../common/timeFormatter/timeFormatter';
import { ViewUtils } from '../../../../common';
import { Task, TaskStatus, JiraTicket } from '../../../../model';
import { DropdownMenuSettings, MenuItem } from 'argo-ui-lib/src/components';
import { JobsService } from '../../jobs.service';
import { JiraService, ToolService } from '../../../../services';

@Component({
    selector: 'ax-job',
    templateUrl: './job.html',
    styles: [ require('./job.scss') ],
})
export class JobComponent implements OnInit, OnDestroy {
    @Input()
    public task: Task;

    @Input()
    public hideLabels: boolean = false;

    @Input()
    public extendedCommiterInfo: boolean = true;

    public jobStartTime: string;
    public labels: string[] = [];
    public isIssuesVisible: boolean = false;
    public jiraIssueLoader: boolean = false;
    public jiraIssues: JiraTicket[] = [];

    private id: string;
    private subscriptions: Subscription[] = [];
    private isJiraConfigured: boolean;

    constructor(private jobsService: JobsService, private jiraService: JiraService, private toolService: ToolService) {
    }

    public ngOnInit() {
        if (!this.hideLabels && this.task.hasOwnProperty('labels')) {
            this.labels = ViewUtils.mapLabelsToList(this.task['labels']);
        }
        this.id = this.task.id;
        this.jobStartTime = TimeFormatter.twelveHoursTime((this.task.launch_time || this.task.create_time) * 1000);
        this.subscriptions.push(this.toolService.isJiraConfigured().subscribe(isConfigured => {
            this.isJiraConfigured = isConfigured;
        }));
        this.subscriptions.push(this.jiraService.jiraIssueCreated.subscribe(info => {
            if (this.task && info.itemId === this.task.id) {
                let issues = this.task.jira_issues || [];
                issues.push(info.issueKey);
                this.task.jira_issues = issues;
                this.loadIssues();
            }
        }));
    }

    public ngOnDestroy() {
        this.subscriptions.forEach(subscription => subscription.unsubscribe());
        this.subscriptions = [];
    }

    public getJobMenu(rootTask: Task): DropdownMenuSettings {
        let menuItems: MenuItem[] = [];
        menuItems.push({
            title: 'Resubmit',
            iconName: 'fa-refresh',
            action: () => this.jobsService.resubmitTask(rootTask)
        });

        // TODO (alexander): Uncomment 'Resubmit Failed' once API support is fixed.
        // if (TaskStatus.Failed === rootTask.status) {
        //     menuItems.push({
        //         title: 'Resubmit Failed',
        //         iconName: 'fa-refresh',
        //         action: () => this.jobsService.resubmitTask(rootTask, true)
        //     });
        // }

        if ([TaskStatus.Cancelled, TaskStatus.Canceling, TaskStatus.Failed, TaskStatus.Success].indexOf(rootTask.status) === -1) {
            menuItems.push({
                title: 'Cancel',
                iconName: 'fa-remove',
                action: () => this.jobsService.cancelTask(rootTask.id, rootTask.name)
            });
        }

        if (this.isJiraConfigured) {
            menuItems.push({
                title: 'Create JIRA Issue',
                iconName: 'ax-icon-jira',
                action: () => this.jiraService.showJiraIssueCreatorPanel.emit({
                    isVisible: true,
                    associateWith: 'service',
                    itemId: this.task.id,
                    name: this.task.name,
                    itemUrl: `${location.protocol}//${location.host}/app/timeline/jobs/${this.task.id}`}),
            });
        }

        let actionMenu: DropdownMenuSettings = new DropdownMenuSettings(menuItems);
        actionMenu.icon = 'fa-ellipsis-v';
        return actionMenu;
    }

    public isCancelling(status: number): boolean {
        return status === TaskStatus.Canceling;
    }

    public async onToggleIssues(event) {
        this.isIssuesVisible = !this.isIssuesVisible;
        this.loadIssues();
    }

    private async loadIssues() {
        if (this.isIssuesVisible) {
            this.jiraIssueLoader = true;
            try {
                await this.jiraService.getJiraIssues({keys: this.task.jira_issues}).then((data: JiraTicket[]) => {
                    this.jiraIssues = data;
                });
            } finally {
                this.jiraIssueLoader = false;
            }
        }
    }
}
