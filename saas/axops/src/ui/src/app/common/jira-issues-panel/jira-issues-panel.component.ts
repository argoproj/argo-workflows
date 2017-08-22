import { Component } from '@angular/core';

import { JiraService } from '../../services';
import { Task, JiraTicket, Application, Deployment } from '../../model';

@Component({
    selector: 'ax-jira-issues-panel',
    templateUrl: './jira-issues-panel.html',
    styles: [ require('./jira-issues-panel.scss') ],
})
export class JiraIssuesPanelComponent {

    public isVisibleJiraIssuesPanel: boolean = false;
    public jiraIssueLoader: boolean = false;
    public itemUrl: string;
    public itemId: string;
    public name: string;
    public associateWith: 'application' | 'service' | 'deployment';
    public jiraIssues: JiraTicket[] = [];
    set item(value: Task | Application) {
        this.itemId = value ? value.id : null;
        this.name = value ? value.name : null;
        if (value && value.jira_issues) {
            this.getJiraIssues(value);
        }
    };

    constructor(private jiraService: JiraService) {
    }

    public closeJiraIssuesPanel() {
        this.jiraService.showJiraIssuesListPanel.emit({isVisible: false});
        this.jiraIssues = [];
    }

    public openJiraCreationPanel() {
        if (this.itemId) {
            this.jiraService.showJiraIssueCreatorPanel.emit({isVisible: true, associateWith: this.associateWith, itemId: this.itemId, name: this.name, itemUrl: this.itemUrl});
        }
    }

    public async getJiraIssues(item: Task | Application | Deployment) {
        let issues = item.jira_issues;
        if (item instanceof Application) {
            issues = item.allJiraIssues;
        }
        this.jiraIssueLoader = true;
        try {
            this.jiraIssues = await this.jiraService.getJiraIssues({keys: issues});
        } finally {
            this.jiraIssueLoader = false;
        }
    }
}
