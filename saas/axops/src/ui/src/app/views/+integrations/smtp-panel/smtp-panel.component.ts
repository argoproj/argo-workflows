import {Component, Input, Output, EventEmitter} from '@angular/core';
import {FormGroup, FormControl, Validators} from '@angular/forms';

import {CustomRegex} from '../../../common/customValidators/CustomRegex';
import {ToolService, NotificationService, ModalService} from '../../../services';
import {ITool, NotificationTool} from '../../../model';

@Component({
    selector: 'ax-smtp-panel',
    templateUrl: './smtp-panel.html',
    styles: [ require('../panels.scss') ]
})
export class SmtpPanelComponent {
    @Output()
    created: EventEmitter<ITool> = new EventEmitter<ITool>();
    @Output()
    deleted: EventEmitter<any> = new EventEmitter<any>();

    private notificationTrackerForm: FormGroup;
    private localType: string = 'smtp';
    private showLoader: boolean = false;
    private wrongCredentialVisible: boolean = false;
    private isCredentialsConfirmed: boolean = false;
    private isCredentialsLoaderVissible: boolean = false;
    private isUserCredentialsVisible: boolean = false;
    private smtpSource: ITool = new NotificationTool();
    private submitted: boolean = false;

    constructor(private toolService: ToolService,
                private notificationService: NotificationService,
                private modalService: ModalService) {
    this.notificationTrackerForm = new FormGroup({
            type: new FormControl(this.localType),
            nickname: new FormControl(this.smtpSource.nickname, Validators.required),
            url: new FormControl(this.smtpSource.url, Validators.required),
            admin_address: new FormControl(this.smtpSource.admin_address, [Validators.required, Validators.pattern(CustomRegex.email)]),
            port: new FormControl(this.smtpSource.port),
            timeout: new FormControl(this.smtpSource.timeout),
            use_tls: new FormControl(this.smtpSource.use_tls),
            username: new FormControl(this.smtpSource.username),
            password: new FormControl('')
        });

        this.notificationTrackerForm.valueChanges.subscribe(data => {
            this.wrongCredentialVisible = false;
        });
    }

    get isAccountConnected(): boolean {
        return this.smtpSource.url !== '';
    }

    @Input()
    set source(val: ITool) {
        this.smtpSource = val || new NotificationTool();
        this.submitted = false;
        this.notificationTrackerForm.reset({
            type: this.localType,
            nickname: this.smtpSource.nickname,
            url: {value: this.smtpSource.url, disabled: this.isAccountConnected},
            admin_address: this.smtpSource.admin_address,
            port: this.smtpSource.port,
            timeout: this.smtpSource.timeout,
            use_tls: this.smtpSource.use_tls,
            username: this.smtpSource.username,
            password: this.smtpSource.password,
        });
        this.isUserCredentialsVisible = !!(this.smtpSource.password || this.smtpSource.username);
    }

    getToolByIdAsync(toolId) {
        this.toolService.getToolByIdAsync(toolId).subscribe(
            success => {
                this.smtpSource = success;
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
                        {message: `${this.localType} account: ${tool.nickname} was successfully connected.`});
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
            id: this.smtpSource.id,
            type: notificationTrackerForm.controls.type.value,
            nickname: notificationTrackerForm.controls.nickname.value,
            url: notificationTrackerForm.controls.url.value,
            admin_address: notificationTrackerForm.controls.admin_address.value,
            use_tls: notificationTrackerForm.controls.use_tls.value === null ? false : notificationTrackerForm.controls.use_tls.value
        };

        tool = this.addPropertyIfValueExist(tool, notificationTrackerForm.controls.port.value, 'port');
        tool = this.addPropertyIfValueExist(tool, notificationTrackerForm.controls.timeout.value, 'timeout');
        tool = this.addPropertyIfValueExist(tool, notificationTrackerForm.controls.username.value, 'username');
        tool = this.addPropertyIfValueExist(tool, notificationTrackerForm.controls.password.value, 'password');

        tool = this.cleanCredentialsIfNotVissible(tool);

        return tool;
    }

    saveChanges(notificationTrackerForm) {
        this.submitted = true;

        if (notificationTrackerForm.valid) {
            let tool: ITool = this.prepareTool(notificationTrackerForm);

            this.toolService.updateToolAsync(tool).subscribe(
                success => {
                    this.notificationService.showNotification.emit(
                        {message: `${this.localType} account: ${tool.nickname} was successfully updated.`});

                    this.getToolByIdAsync(success.id);
                }
            );
        }
    }

    disconnectAccount() {
        this.modalService.showModal(`Disconnecting ${this.localType}`,
            `You are sure you want to disconnect ${this.localType} connection?`).subscribe(result => {
            if (result) {
                this.toolService.deleteToolAsync(this.smtpSource.id).subscribe(
                    success => {
                        this.notificationService.showNotification.emit(
                            {message: `${this.localType} account: ${this.smtpSource.username} was successfully disconnect.`});
                        this.deleted.emit({});
                    }
                );
            }
        });
    }

    cleanCredentialsIfNotVissible(tool) {
        if (!this.isUserCredentialsVisible) {
            delete tool.username;
            delete tool.password;
        }
        return tool;
    }

    addPropertyIfValueExist(object, value, property) {
        if (value) {
            object[property] = value;
        }
        return object;
    }
}
