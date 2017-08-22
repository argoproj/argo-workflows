import { Subscription } from 'rxjs';
import { Component, Input, Output, EventEmitter, OnInit, OnDestroy } from '@angular/core';
import { Router } from '@angular/router';

import { Application } from '../../../model';
import { ApplicationsService, JiraService, ToolService } from '../../../services';

@Component({
    selector: 'ax-application-panel',
    templateUrl: './application-panel.html',
    styles: [ require('./application-panel.scss') ],
})
export class ApplicationPanelComponent implements OnInit, OnDestroy {
    @Input()
    public get application(): Application {
        return this.appData;
    }

    public set application(appData: Application) {
        this.appData = appData;
    }

    @Output()
    public onSelectApplication: EventEmitter<any> = new EventEmitter();

    public isJiraConfigured: boolean = false;
    private subscriptions: Subscription[] = [];
    private appData: Application;

    constructor(private router: Router, private applicationService: ApplicationsService, private jiraService: JiraService, private toolService: ToolService) {
    }

    public ngOnInit() {
        this.subscriptions.push(this.toolService.isJiraConfigured().subscribe(isConfigured => this.isJiraConfigured = isConfigured));
        this.subscriptions.push(this.jiraService.jiraIssueCreated.subscribe((info: { itemId: string, itemType: string, issueKey: string }) => {
            if (info.itemType === 'application' && info.itemId === this.application.id) {
                let issues = (this.application.jira_issues || []);
                issues.push(info.issueKey);
                this.application.jira_issues = issues;
            }
        }));
    }

    public ngOnDestroy() {
        this.subscriptions.forEach(item => item.unsubscribe());
        this.subscriptions = [];
    }

    public toggleIssuesPanel(application: Application) {
        this.jiraService.showJiraIssuesListPanel.emit(
            { isVisible: true, associateWith: 'application', item: application, itemUrl: `${location.protocol}//${location.host}/app/applications/details/${application.id}`});
    }
}
