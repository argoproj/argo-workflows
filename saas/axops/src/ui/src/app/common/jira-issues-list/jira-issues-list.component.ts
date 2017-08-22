import { Subscription } from 'rxjs';
import { Component, Input, OnInit, OnDestroy } from '@angular/core';

import { JiraTicket, ITool } from '../../model';
import { ToolService } from '../../services';

@Component({
    selector: 'ax-jira-issues-list',
    templateUrl: './jira-issues-list.html',
    styles: [ require('./jira-issues-list.scss') ],
})
export class JiraIssuesListComponent implements OnInit, OnDestroy {
    @Input()
    public source: JiraTicket[] = [];

    @Input()
    public customClass: string;

    public baseJiraURL = '';

    private subscriptions: Subscription[] = [];

    constructor(private toolService: ToolService) {}

    public ngOnInit() {
        this.subscriptions.push(this.toolService.getJiraConfig().subscribe((config: ITool) => {
            this.baseJiraURL = config ? config.url : '';
        }));
    }

    public ngOnDestroy() {
        this.subscriptions.forEach(subscription => subscription.unsubscribe());
        this.subscriptions = [];
    }
}
