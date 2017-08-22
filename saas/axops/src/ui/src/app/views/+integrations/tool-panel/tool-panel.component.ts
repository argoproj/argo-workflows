import {Component, Output, Input, EventEmitter} from '@angular/core';
import {FormGroup, FormControl, Validators} from '@angular/forms';

import {CustomRegex} from '../../../common/customValidators/CustomRegex';
import {Tool, ITool} from '../../../model';
import {ToolService, NotificationService, ModalService} from '../../../services';
import {SortOperations} from '../../../common/sortOperations/sortOperations';
import { SelectOption } from 'argo-ui-lib/src/components/multi-select/multi-select.component';

@Component({
    selector: 'ax-tool-panel',
    templateUrl: './tool-panel.html',
    styles: [ require('../panels.scss') ],
})

export class ToolPanelComponent {
    @Output()
    created: EventEmitter<Tool> = new EventEmitter<Tool>();
    @Output()
    deleted: EventEmitter<any> = new EventEmitter<any>();

    public submitted: boolean;
    public isAccountConnected: boolean;
    public allRepos: SelectOption[] = [];

    private toolForm: FormGroup;
    private toolSource: ITool = new Tool();
    private showLoader: boolean = false;
    private wrongCredentialVissible: boolean = false;
    private isCredentialsConfirmed: boolean = false;

    constructor(private toolService: ToolService,
                private notificationService: NotificationService,
                private modalService: ModalService) {
        this.toolForm = new FormGroup({
            type: new FormControl(''),
            username: new FormControl(this.toolSource.username, [Validators.required, Validators.pattern(CustomRegex.myToolsUsername)]),
            password: new FormControl(this.toolSource.password),
            repos: new FormControl(this.toolSource.hasOwnProperty('repos') ? this.toolSource['repos'] : [], (c: FormControl) => {
                if (!this.isAccountConnected || (c.value && c.value.length > 0)) {
                    return null;
                }
                return {
                    repos: {
                        valid: false,
                    },
                };
            }),
            all_repos: new FormControl(this.toolSource['all_repos']),
            use_webhook: new FormControl(!!this.toolSource.use_webhook),
        });

        this.initialization();
    }

    initialization() {
        this.toolForm.valueChanges.subscribe(data => {
            this.wrongCredentialVissible = false;
            this.isCredentialsConfirmed = false;
        });
    }

    @Input()
    set type(val: string) {
        this.toolForm.controls['type'].setValue(val);
    }

    get type(): string {
        return this.toolForm.controls['type'].value;
    }

    @Input()
    set source(value) {
        this.submitted = false;
        this.toolSource = value || new Tool();
        this.toolSource['all_repos'] = this.toolSource.hasOwnProperty('all_repos') &&
            this.toolSource['all_repos'].length ? SortOperations.sortNoCaseSensitive(this.toolSource['all_repos']) : [];
        let type = this.toolForm.controls['type'].value;
        this.toolForm.reset();
        this.toolForm.controls['type'].setValue(type);
        this.toolForm.controls['username'].setValue(this.toolSource.username);
        this.toolForm.controls['password'].setValue(this.toolSource.password);
        this.toolForm.controls['repos'].setValue(this.toolSource.repos || []);
        this.toolForm.controls['all_repos'].setValue(this.toolSource.all_repos || []);
        this.toolForm.controls['use_webhook'].setValue(!!this.toolSource.use_webhook);

        this.allRepos = this.parseRepoUrls(this.toolSource.all_repos || []);

        if (value) {
            this.testCredentials(this.toolForm);
        }
    }

    getToolByIdAsync(toolId) {
        this.toolService.getToolByIdAsync(toolId).subscribe(
            success => {
                this.toolSource = success;

                this.initialization();
            }
        );
    }

    connectAccount(toolForm) {
        this.submitted = true;
        if (toolForm.valid) {
            this.showLoader = true;
            let tool: ITool = {
                type: toolForm.controls.type.value,
                username: toolForm.controls.username.value,
                password: toolForm.controls.password.value,
                use_webhook: toolForm.controls.use_webhook.value,
            };

            this.toolService.connectAccountAsync(tool).subscribe(
                success => {
                    this.notificationService.showNotification.emit(
                        {message: `${this.type} account: ${tool.username} was successfully connected.`});
                    this.created.emit(success);
                    this.showLoader = false;
                    this.isAccountConnected = true;

                    this.getToolByIdAsync(success.id);
                }, error => {
                    this.showLoader = false;
                    this.wrongCredentialVissible = true;
                });
        }
    }

    saveChanges(toolForm) {
        let tool: any = {
            id: this.toolSource.id,
            type: toolForm.controls.type.value,
            username: toolForm.controls.username.value,
            password: toolForm.controls.password.value,
            all_repos: toolForm.controls.all_repos.value,
            repos: toolForm.controls.repos.value,
            use_webhook: toolForm.controls.use_webhook.value,
        };

        this.toolService.updateToolAsync(tool).subscribe(
            success => {
                this.created.emit(success);
                this.notificationService.showNotification.emit(
                    {message: `${this.type} account: ${tool.username} was successfully updated.`});
            }
        );
    }

    disconnectAccount() {
        this.modalService.showModal(`Disconnecting ${this.type}`,
            `You are sure you want to disconnect ${this.type} connection?`).subscribe(result => {
            if (result) {
                this.toolService.deleteToolAsync(this.toolSource.id).subscribe(
                    success => {
                        this.deleted.emit({});
                        this.notificationService.showNotification.emit(
                            {message: `${this.type} account: ${this.toolSource.username} was successfully disconnect.`});
                    }
                );
            }
        });
    }

    hasWebHookSupport(type: string): boolean {
        return type === 'bitbucket' || type === 'github';
    }

    private parseRepoUrls(repos: string[]): SelectOption[] {
        return repos.map(repo => {
            return {
                value: repo,
                name: repo.replace(/^.*\/\/[^\/]+/, '')
            };
        });
    }

    private testCredentials(form) {
        this.wrongCredentialVissible = false;

        let tool: ITool = {
            id: this.toolSource.id,
            type: this.toolSource.type,
            username: form.controls.username.value,
            password: form.controls.password.value,
            use_webhook: form.controls.use_webhook.value,
        };

        this.toolService.testCredentialsAsync(tool).subscribe(() => {
            this.isAccountConnected = true;
        }, () => {
            this.wrongCredentialVissible = true;
        });
    }
}
