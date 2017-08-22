import { Component, OnInit, OnDestroy } from '@angular/core';
import { FormGroup, Validators, FormControl } from '@angular/forms';
import { Subscription } from 'rxjs';

import { NotificationsService } from 'argo-ui-lib/src/components';
import { JiraProject, JiraIssueResponse, ITool } from '../../model';
import { JiraService, ToolService, AuthenticationService } from '../../services';

@Component({
    selector: 'ax-jira-issue-creator-panel',
    templateUrl: './jira-issue-creator-panel.html',
    styles: [ require('./jira-issue-creator-panel.scss') ],
})
export class JiraIssueCreatorPanelComponent implements OnInit, OnDestroy {

    public jiraIssueForm: FormGroup;
    public isVisibleJiraProjectSelectorPanel: boolean = false;
    public serviceId: string;
    public associateWith: 'application' | 'service' | 'deployment';
    public jiraProjects: JiraProject[] = [];
    public jiraProjectsLoader: boolean = false;
    public jiraProjectSettings: {name: string, value: string}[] = [];
    public jiraIssueTypeSettings: {name: string, value: string}[] = [];
    public selectedProject: JiraProject;
    public selectedIssueType: {name: string, id: number} = {name: null, id: null};
    public isSubmitClicked: boolean = false;
    public set name(value: string) {
        if (value) {
            this.jiraIssueForm.controls['summary'].setValue(value);
        }
    }
    public set itemUrl(value: string) {
        if (value) {
            this.jiraIssueForm.controls['description'].setValue(value);
        }
    }

    private subscriptions: Subscription[] = [];

    constructor(private jiraService: JiraService,
                private toolService: ToolService,
                private authenticationService: AuthenticationService,
                private notificationsService: NotificationsService) {
    }

    public async ngOnInit() {
        this.initForm();
        this.subscriptions.push(this.toolService.getJiraConfig().subscribe(config => {
            this.getJiraProjects(config);
        }));
    }

    public ngOnDestroy() {
        this.subscriptions.forEach(subscription => subscription.unsubscribe());
        this.subscriptions = [];
    }

    public async getJiraProjects(config: ITool) {
        this.jiraProjects = [];
        this.jiraProjectSettings = [];
        if (config) {
            try {
                this.jiraProjectsLoader = true;
                this.jiraProjects = (await this.jiraService.getJiraProjects()).filter(project => (config.projects || []).indexOf(project.key) > -1);
                this.jiraProjectSettings = this.prepareJiraProjectsActionMenu(this.jiraProjects);
                this.jiraProjectsLoader = false;
                this.jiraIssueForm.controls['project'].setValue(this.jiraProjectSettings && this.jiraProjectSettings.length ? this.jiraProjectSettings[0].value : '');
            } catch (e) {
                this.jiraProjectsLoader = false;
            }
        }
    }

    public prepareJiraProjectsActionMenu(jiraProjects: JiraProject[]): {name: string, value: string}[] {
        let menuItems: {name: string, value: string}[] = [];

        jiraProjects.forEach(project => {
            menuItems.push({
                name: project.name,
                value: project.key,
            });
        });

        return menuItems;
    }

    public closeJiraProjectSelectorPanel() {
        this.isSubmitClicked = false;
        this.selectedProject = null;
        this.selectedIssueType = {name: null, id: null};
        this.jiraService.showJiraIssueCreatorPanel.emit({isVisible: false});
        // Reset form values after issue is created.
        this.initForm();
    }

    public async createJiraIssue() {
        this.isSubmitClicked = true;
        if (this.jiraIssueForm.valid) {
            let jiraIssueResponse: JiraIssueResponse = await this.jiraService.createJiraIssue(this.jiraIssueForm.value);

            let itemId = this.serviceId;
            let itemType = this.associateWith;
            await this.jiraService.associateJiraIssueWith(jiraIssueResponse.key, itemId, itemType);

            if (!this.jiraIssueForm.value.createAnother) {
                this.closeJiraProjectSelectorPanel();
            }

            this.jiraIssueForm.controls['summary'].reset();
            this.jiraIssueForm.controls['description'].reset();
            this.jiraService.jiraIssueCreated.emit({ itemId, itemType, issueKey: jiraIssueResponse.key });
            this.notificationsService.success(`The JIRA issue was successfully created.`);
            // Reset form values after issue is created.
            this.initForm();
        }
    }

    private initForm() {
        this.jiraIssueForm = new FormGroup({
            project: new FormControl('', Validators.required),
            issuetype: new FormControl('Bug', Validators.required),
            summary: new FormControl('', Validators.required),
            reporter: new FormControl(this.authenticationService.getUser().username, Validators.required),
            description: new FormControl(''),
            createAnother: new FormControl(false),
        });

        this.jiraIssueTypeSettings = [
            { name: 'Story', value: 'Story' },
            { name: 'Task', value: 'Task' },
            { name: 'Bug', value: 'Bug' },
        ];
    }
}
