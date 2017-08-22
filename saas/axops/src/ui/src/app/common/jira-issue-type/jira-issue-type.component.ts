import { Component, Input } from '@angular/core';

import { JIRA_ISSUE_TYPE } from '../../model';

@Component({
    selector: 'ax-jira-issue-type',
    templateUrl: './jira-issue-type.html',
    styles: [ require('./jira-issue-type.scss') ],
})
export class JiraIssueTypeComponent {
    @Input()
    public set type(value) {
        switch (value) {
            case JIRA_ISSUE_TYPE.BUG:
                this.icon = 'ax-icon-tag';
                this.issueType = 'bug';
                break;
            case JIRA_ISSUE_TYPE.TASK:
                this.icon = 'ax-icon-tag';
                this.issueType = 'task';
                break;
            case JIRA_ISSUE_TYPE.STORY:
                this.icon = 'ax-icon-tag';
                this.issueType = 'story';
                break;
            case JIRA_ISSUE_TYPE.EPIC:
                this.icon = 'ax-icon-tag';
                this.issueType = 'epic';
                break;
            case JIRA_ISSUE_TYPE.SUBTASK:
                this.icon = 'ax-icon-tag';
                this.issueType = 'sub-task';
                break;
            default:
                this.icon = '';
        }
    };

    public issueType: string;
    public icon: string;
}
