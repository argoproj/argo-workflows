import {Component, Input, Output, EventEmitter} from '@angular/core';
import {FormGroup, FormControl, Validators} from '@angular/forms';

import {ToolService, NotificationService, ModalService} from '../../../services';
import {ITool, SlackTool} from '../../../model';

@Component({
    selector: 'ax-slack-panel',
    templateUrl: './slack-panel.html',
    styles: [ require('../panels.scss') ]
})
export class SlackPanelComponent {
    @Output()
    created: EventEmitter<ITool> = new EventEmitter<ITool>();
    @Output()
    deleted: EventEmitter<any> = new EventEmitter<any>();

    private slackForm: FormGroup;
    private localType: string = 'slack';
    private showLoader: boolean = false;
    private wrongCredentialVisible: boolean = false;
    private isCredentialsConfirmed: boolean = false;
    private isCredentialsLoaderVissible: boolean = false;
    private dataSource: ITool = new SlackTool();
    private submitted: boolean = false;

    constructor(private toolService: ToolService,
                private notificationService: NotificationService,
                private modalService: ModalService) {
        this.slackForm = new FormGroup({
            type: new FormControl(this.localType),
            oauth_token: new FormControl(this.dataSource.oauth_token, Validators.required),
        });

        this.slackForm.valueChanges.subscribe(data => {
            this.wrongCredentialVisible = false;
        });
    }

    get isAccountConnected(): boolean {
        return this.dataSource.oauth_token !== '';
    }

    @Input()
    set source(val: ITool) {
        this.dataSource = val || new SlackTool();
        this.submitted = false;
        this.slackForm.reset({
            type: this.localType,
            oauth_token: this.dataSource.oauth_token,
        });
    }

    getToolByIdAsync(toolId) {
        this.toolService.getToolByIdAsync(toolId).subscribe(
            success => {
                this.dataSource = success;
            }
        );
    }

    connectAccount(notificationTrackerForm) {
        this.submitted = true;

        if (notificationTrackerForm.valid) {
            this.showLoader = true;
            let tool: ITool = this.prepareTool(notificationTrackerForm);

            this.toolService.connectAccountAsync(tool).subscribe(
                success => {
                    this.notificationService.showNotification.emit(
                        {message: `${this.localType} account was successfully connected.`});
                    this.showLoader = false;
                    this.created.emit(success);
                }, error => {
                    this.showLoader = false;
                    this.wrongCredentialVisible = true;
                });
        }
    }

    testCredentials(notificationTrackerForm, isConnected = false) {
        this.isCredentialsLoaderVissible = true;
        this.wrongCredentialVisible = false;

        let tool: ITool = this.prepareTool(notificationTrackerForm);

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

    prepareTool(notificationTrackerForm): ITool {
        let tool: ITool = {
            id: this.dataSource.id,
            type: notificationTrackerForm.controls.type.value,
            oauth_token: notificationTrackerForm.controls.oauth_token.value,
        };

        return tool;
    }

    saveChanges(notificationTrackerForm) {
        this.submitted = true;

        if (notificationTrackerForm.valid) {
            let tool: ITool = this.prepareTool(notificationTrackerForm);

            this.toolService.updateToolAsync(tool).subscribe(
                success => {
                    this.notificationService.showNotification.emit(
                        {message: `${this.localType} account was successfully updated.`});

                    this.getToolByIdAsync(success.id);
                }
            );
        }
    }

    disconnectAccount() {
        this.modalService.showModal(`Disconnecting ${this.localType}`,
            `You are sure you want to disconnect ${this.localType} connection?`).subscribe(result => {
            if (result) {
                this.toolService.deleteToolAsync(this.dataSource.id).subscribe(
                    success => {
                        this.notificationService.showNotification.emit(
                            {message: `${this.localType} account was successfully disconnect.`});
                        this.deleted.emit({});
                    }
                );
            }
        });
    }
}
