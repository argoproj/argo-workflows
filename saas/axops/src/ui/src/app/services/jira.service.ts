import { EventEmitter, Injectable, Output } from '@angular/core';
import { Http, URLSearchParams } from '@angular/http';

import { AxHeaders } from './headers';
import { Task, JiraIssue, JiraTicket, JiraProject, Application } from '../model';

@Injectable()
export class JiraService {
    @Output()
    public showJiraIssueCreatorPanel:
        EventEmitter<{ isVisible: boolean, associateWith?: 'application' | 'service' | 'deployment', itemId?: string, name?: string, itemUrl?: string }> = new EventEmitter();
    @Output()
    public showJiraIssuesListPanel:
        EventEmitter<{ isVisible: boolean, associateWith?: 'application' | 'service' | 'deployment', item?: Task | Application, itemUrl?: string }> = new EventEmitter();
    @Output()
    public jiraIssueCreated: EventEmitter<{ itemId: string, itemType: string, issueKey: string }> = new EventEmitter();

    constructor(private http: Http) {
    }

    public async getJiraIssues(params: {
        project?: string,
        status?: string,
        component?: string,
        labels?: string,
        issuetype?: string,
        priority?: string,
        creator?: string,
        assignee?: string,
        reporter?: string,
        fixversion?: string,
        affectedversion?: string,
        keys?: string[],
    }, hideLoader = true): Promise<JiraTicket[]> {
        let filter = new URLSearchParams();

        if (params.project) {
            filter.set('project', params.project.toString());
        }

        if (params.status) {
            filter.set('status', params.status.toString());
        }

        if (params.component) {
            filter.set('component', params.component.toString());
        }

        if (params.labels) {
            filter.set('labels', params.labels.toString());
        }

        if (params.issuetype) {
            filter.set('issuetype', params.issuetype.toString());
        }

        if (params.priority) {
            filter.set('priority', params.priority.toString());
        }

        if (params.creator) {
            filter.set('creator', params.creator.toString());
        }

        if (params.assignee) {
            filter.set('assignee', params.assignee.toString());
        }

        if (params.reporter) {
            filter.set('reporter', params.reporter.toString());
        }

        if (params.fixversion) {
            filter.set('fixversion', params.fixversion.toString());
        }

        if (params.affectedversion) {
            filter.set('affectedversion', params.affectedversion.toString());
        }

        if (params.keys) {
            filter.set('ids', params.keys.toString());
        }

        return this.http.get(`v1/jira/issues`, {headers: new AxHeaders({ noLoader: hideLoader }), search: filter}).toPromise().then(res => res.json());
    }

    public async getJiraProjects(): Promise<JiraProject[]> {
        return this.http.get(`v1/jira/projects`, {headers: new AxHeaders({ noLoader: true })}).toPromise().then(res => res.json().data);
    }

    public async getIssueByKey(jiraIssueKey: string): Promise<any> {
        return this.http.get(`v1/jira/issues/${jiraIssueKey}`, {headers: new AxHeaders({ noLoader: true })}).toPromise().then(res => res.json());
    }

    public async createJiraIssue(jiraIssue: JiraIssue): Promise<any> {
        return this.http.post('v1/jira/issues', JSON.stringify(jiraIssue)).toPromise().then(res => res.json());
    }

    public associateJiraIssueWith(jiraId: string, serviceId: string, association: 'service' | 'application' | 'deployment') {
        return this.http.put(`v1/jira/issues/${jiraId}/${association}/${serviceId}`, {}).toPromise().then(res => res.json());
    }
}
