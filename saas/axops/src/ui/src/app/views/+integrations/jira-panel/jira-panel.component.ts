import { Component, Input, Output, EventEmitter } from '@angular/core';
import { FormGroup, FormControl, Validators } from '@angular/forms';

import { CustomRegex } from '../../../common/customValidators/CustomRegex';
import { ITool, JiraTool } from '../../../model';
import { ToolService, NotificationService, ModalService, JiraService } from '../../../services';
import { SelectOption } from 'argo-ui-lib/src/components/multi-select/multi-select.component';

@Component({
    selector: 'ax-jira-panel',
    templateUrl: './jira-panel.html',
})
export class JiraPanelComponent {

    public isCredentialsLoaderVissible: boolean;
    public wrongCredentialVisible: boolean;
    public jiraSource: ITool = new JiraTool();
    public allProjects: SelectOption[] = [];

    @Output()
    public created: EventEmitter<ITool> = new EventEmitter<ITool>();
    @Output()
    public deleted: EventEmitter<any> = new EventEmitter<any>();

    public submitted: boolean;

    private jiraForm: FormGroup;
    private showLoader: boolean = false;
    private isCredentialsConfirmed: boolean = false;
    private wrongCredentialVissible: boolean = false;

    constructor(private toolService: ToolService,
                private jiraService: JiraService,
                private notificationService: NotificationService,
                private modalService: ModalService) {
        this.jiraForm = new FormGroup({
            type: new FormControl('jira'),
            username: new FormControl(this.jiraSource.username, [Validators.pattern(CustomRegex.myToolsUsername), Validators.required]),
            password: new FormControl(this.jiraSource.password),
            url: new FormControl(this.jiraSource.hostname, Validators.required),
            hostname: new FormControl(this.jiraSource.hostname, Validators.required),
            projects: new FormControl(this.jiraSource.hasOwnProperty('projects') ? this.jiraSource['projects'] : []),
        });

        this.initialization();
    }

    get isAccountConnected(): boolean {
        return this.jiraSource.username !== '';
    }

    @Input()
    set source(value: ITool) {
        this.jiraSource = value || new JiraTool();
        this.submitted = false;
        this.jiraForm.reset({
            type: 'jira',
            username: this.jiraSource.username,
            password: this.jiraSource.password,
            url: {value: this.jiraSource.url, disabled: this.isAccountConnected},
            hostname: {value: this.jiraSource.hostname, disabled: this.isAccountConnected},
            projects: this.jiraSource.projects || [],
        });
        if (this.isAccountConnected) {
            this.loadProjects();
        }
    }

    public connectAccount(jiraForm) {
        this.submitted = true;
        if (jiraForm.valid) {
            this.showLoader = true;
            let tool: ITool = {
                type: jiraForm.controls.type.value,
                username: jiraForm.controls.username.value,
                password: jiraForm.controls.password.value,
                hostname: jiraForm.controls.hostname.value,
                projects: jiraForm.controls.projects.value,
                url: jiraForm.controls.url.value,
            };

            this.toolService.connectAccountAsync(tool).subscribe(
                success => {
                    this.showLoader = false;
                    this.notificationService.showNotification.emit({
                        message: `Jira server: ${tool.url} was successfully connected.`});
                    this.created.emit(success);
                    this.loadProjects();
                }, error => {
                    this.showLoader = false;
                    this.wrongCredentialVissible = true;
                });
        }
    }

    public saveChanges(jiraForm) {
        this.submitted = true;
        if (jiraForm.valid) {
            let tool: ITool = {
                id: this.jiraSource.id,
                type: jiraForm.controls.type.value,
                username: jiraForm.controls.username.value,
                password: jiraForm.controls.password.value,
                hostname: jiraForm.controls.hostname.value,
                projects: jiraForm.controls.projects.value,
                url: jiraForm.controls.url.value,
            };

            this.submitted = false;
            this.toolService.updateToolAsync(tool).subscribe(
                success => {
                    this.created.emit(success);
                    this.notificationService.showNotification.emit({message: `Jira server: ${tool.url} was successfully updated.`});
                }
            );
        }
    }

    public disconnectAccount() {
        this.modalService.showModal('Disconnecting Jira', 'You are sure you want to disconnect Jira account?').subscribe(result => {
            if (result) {
                this.toolService.deleteToolAsync(this.jiraSource.id).subscribe(
                    success => {
                        this.deleted.emit({});
                        this.notificationService.showNotification.emit(
                            {message: `Jira server: ${this.jiraSource.url} was successfully disconnected.`});
                    }
                );
            }
        });
    }

    public testCredentials(jiraForm) {
        this.isCredentialsLoaderVissible = true;
        this.wrongCredentialVisible = false;

        let tool: ITool = {
            id: this.jiraSource.id,
            type: jiraForm.controls['type'].value,
            username: jiraForm.controls['username'].value,
            password: jiraForm.controls['password'].value,
            hostname: jiraForm.controls['hostname'].value,
            projects: jiraForm.controls['projects'].value,
            url: jiraForm.controls['url'].value,
        };

        this.toolService.testCredentialsAsync(tool).subscribe(
            success => {
                this.isCredentialsLoaderVissible = false;
                this.isCredentialsConfirmed = true;
            }, error => {
                this.isCredentialsLoaderVissible = false;
                this.isCredentialsConfirmed = false;
                this.wrongCredentialVisible = true;
            });
    }

    private async loadProjects() {
        this.showLoader = true;
        this.allProjects = (await this.jiraService.getJiraProjects()).map(proj => ({
            value: proj.key,
            name: proj.name
        }));
        this.showLoader = false;
    }

    private initialization() {
        this.jiraForm.valueChanges.subscribe(data => {
            this.wrongCredentialVissible = false;
            this.isCredentialsConfirmed = false;
        });
    }
}
