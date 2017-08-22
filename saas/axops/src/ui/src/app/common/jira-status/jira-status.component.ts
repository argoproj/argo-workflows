import { Component, Input } from '@angular/core';

export const JIRA_STATUSES = {
    'OPEN': '1',
    'INPROGRESS': '3',
    'TODO': '10000',
    'DONE': '10001',
};

@Component({
    selector: 'ax-jira-status',
    templateUrl: './jira-status.html',
    styles: [ require('./jira-status.scss') ],
})
export class JiraStatusComponent {
    @Input()
    public set status(value) {
        this.buttonClass = value;
        switch (value) {
            case JIRA_STATUSES.OPEN:
                this.buttonLabel = 'open';
                this.buttonClass = 'open';
                break;
            case JIRA_STATUSES.TODO:
                this.buttonLabel = 'to do';
                this.buttonClass = 'to-do';
                break;
            case JIRA_STATUSES.INPROGRESS:
                this.buttonLabel = 'in progress';
                this.buttonClass = 'in-progress';
                break;
            case JIRA_STATUSES.DONE:
                this.buttonLabel = 'done';
                this.buttonClass = 'done';
                break;
            default:
                this.buttonLabel = `Unknown status: ${ value }`;
        }
    };

    public buttonLabel: string;
    public buttonClass: string;
}
