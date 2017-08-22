import { Component, Input, Output, EventEmitter } from '@angular/core';
import { FormGroup, FormControl, Validators } from '@angular/forms';

import { CustomRegex } from '../../../common/customValidators/CustomRegex';
import { NexusTool, ITool } from '../../../model';
import { ToolService, NotificationService, ModalService } from '../../../services';

@Component({
    selector: 'ax-nexus-panel',
    templateUrl: './nexus-panel.html',
    styles: [ require('../panels.scss') ]
})
export class NexusPanelComponent {

    isCredentialsLoaderVissible: boolean;
    wrongCredentialVisible: boolean;
    nexusSource: ITool = new NexusTool();

    @Output()
    created: EventEmitter<ITool> = new EventEmitter<ITool>();
    @Output()
    deleted: EventEmitter<any> = new EventEmitter<any>();

    public submitted: boolean;

    private nexusForm: FormGroup;
    private showLoader: boolean = false;
    private isCredentialsConfirmed: boolean = false;
    private wrongCredentialVissible: boolean = false;

    constructor(private toolService: ToolService,
                private notificationService: NotificationService,
                private modalService: ModalService) {
        this.nexusForm = new FormGroup({
            type: new FormControl('nexus'),
            username: new FormControl(this.nexusSource.username, [Validators.pattern(CustomRegex.myToolsUsername), Validators.required]),
            password: new FormControl(this.nexusSource.password),
            hostname: new FormControl(this.nexusSource.hostname, Validators.required),
            port: new FormControl(this.nexusSource.port, Validators.required),
        });

        this.initialization();
    }

    initialization() {
        this.nexusForm.valueChanges.subscribe(data => {
            this.wrongCredentialVissible = false;
            this.isCredentialsConfirmed = false;
        });
    }

    get isAccountConnected(): boolean {
        return this.nexusSource.username !== '';
    }

    @Input()
    set source(value: ITool) {
        this.nexusSource = value || new NexusTool();
        this.submitted = false;
        this.nexusForm.reset({
            type: 'nexus',
            username: this.nexusSource.username,
            password: this.nexusSource.password,
            hostname: {value: this.nexusSource.hostname, disabled: this.isAccountConnected},
            port: {value: this.nexusSource.port, disabled: this.isAccountConnected},
        });
    }

    connectAccount(nexusForm) {
        this.submitted = true;
        if (nexusForm.valid) {
            this.showLoader = true;
            let tool: ITool = {
                type: nexusForm.controls.type.value,
                username: nexusForm.controls.username.value,
                password: nexusForm.controls.password.value,
                hostname: nexusForm.controls.hostname.value,
                port: parseInt(nexusForm.controls.port.value, 10)
            };

            this.toolService.connectAccountAsync(tool).subscribe(
                success => {
                    this.showLoader = false;
                    this.notificationService.showNotification.emit({
                        message: `Nexus repository: ${tool.hostname} was successfully connected.`});
                    this.created.emit(success);
                }, error => {
                    this.showLoader = false;
                    this.wrongCredentialVissible = true;
                });
        }

    }

    saveChanges(nexusForm) {
        this.submitted = true;
        let tool: ITool = {
            id: this.nexusSource.id,
            type: nexusForm.controls.type.value,
            username: nexusForm.controls.username.value,
            password: nexusForm.controls.password.value,
            hostname: nexusForm.controls.hostname.value,
            port: parseInt(nexusForm.controls.port.value, 10),
        };

        this.toolService.updateToolAsync(tool).subscribe(
            success => {
                this.created.emit(success);
                this.notificationService.showNotification.emit({message: `Nexus repository: ${tool.hostname} was successfully updated.`});
            }
        );
    }

    disconnectAccount() {
        this.modalService.showModal('Disconnecting Nexus', 'You are sure you want to disconnect Nexus repository?').subscribe(result => {
            if (result) {
                this.toolService.deleteToolAsync(this.nexusSource.id).subscribe(
                    success => {
                        this.deleted.emit({});
                        this.notificationService.showNotification.emit(
                            {message: `Nexus repository: ${this.nexusSource.hostname} was successfully disconnected.`});
                    }
                );
            }
        });
    }

    testCredentials(nexusForm) {
        this.isCredentialsLoaderVissible = true;
        this.wrongCredentialVisible = false;

        let tool: ITool = {
            id: this.nexusSource.id,
            type: nexusForm.controls['type'].value,
            username: nexusForm.controls['username'].value,
            password: nexusForm.controls['password'].value,
            hostname: nexusForm.controls['hostname'].value,
            port: parseInt(nexusForm.controls['port'].value, 10),
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

}
