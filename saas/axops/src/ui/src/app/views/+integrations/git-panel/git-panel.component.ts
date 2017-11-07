import {Component, Input, Output, EventEmitter, OnInit} from '@angular/core';
import {FormGroup, FormControl, Validators} from '@angular/forms';

import {CustomRegex} from '../../../common/customValidators/CustomRegex';
import {Tool, ITool} from '../../../model';
import {ToolService, NotificationService, ModalService} from '../../../services';

@Component({
    selector: 'ax-git-panel',
    templateUrl: './git-panel.html',
    styles: [ require('../panels.scss') ]
})
export class GitPanelComponent implements OnInit {
    gitSource: Tool = new Tool();
    @Output()
    created: EventEmitter<Tool> = new EventEmitter<Tool>();
    @Output()
    deleted: EventEmitter<any> = new EventEmitter<any>();

    public submitted: boolean;
    public isAccountConnected: boolean;

    private gitForm: FormGroup;
    private showLoader: boolean = false;
    private wrongCredentialVissible: boolean = false;

    constructor(private toolService: ToolService,
                private notificationService: NotificationService,
                private modalService: ModalService) {
        this.gitForm = new FormGroup({
            type: new FormControl('git'),
            username: new FormControl(this.gitSource.username),
            password: new FormControl(this.gitSource.password),
            isPrivate: new FormControl(this.gitSource.username !== undefined),
            url: new FormControl(this.gitSource.url, Validators.required),
        });

        this.initialization();
    }

    public ngOnInit() {
        this.gitForm.get('isPrivate').valueChanges.subscribe((isPrivate: boolean) => {
            if (isPrivate) {
                this.gitForm.get('username').setValidators([Validators.required, Validators.pattern(CustomRegex.myToolsUsername)]);
                this.gitForm.get('password').setValidators([Validators.required]);
                this.gitForm.get('username').setValue(this.gitSource.username);
            } else {
                this.gitForm.get('username').clearValidators();
                this.gitForm.get('password').clearValidators();
                this.gitForm.get('username').setValue(null);
            }

            this.gitForm.get('password').setValue(null);
            this.gitForm.get('username').updateValueAndValidity();
            this.gitForm.get('password').updateValueAndValidity();
        });
    }

    initialization() {
        this.gitForm.valueChanges.subscribe(data => {
            this.wrongCredentialVissible = false;
        });
    }

    @Input()
    set source(value: Tool) {
        this.gitSource = value || new Tool();
        this.submitted = false;
        this.gitForm.reset({
            type: 'git',
            username: this.gitSource.username,
            password: this.gitSource.password,
            isPrivate: this.gitSource.username !== undefined,
            url: {value: this.gitSource.url, disabled: this.isAccountConnected}
        });

        if (value) {
            this.testCredentials(this.gitForm, () => {
                this.isAccountConnected = true;
            });
        }
    }

    connectAccount(gitForm: any) {
        this.submitted = true;
        if (gitForm.valid) {
            this.testCredentials(gitForm, () => {
                this.showLoader = true;

                let tool: ITool = {
                    type: gitForm.controls.type.value,
                    username: gitForm.controls.username.value,
                    password: gitForm.controls.password.value,
                    url: gitForm.controls.url.value
                };

                this.toolService.connectAccountAsync(tool).subscribe(
                    success => {
                        this.showLoader = false;
                        this.notificationService.showNotification.emit({
                            message: `Git repository: ${tool.url} was successfully connected.`});
                        this.created.emit(success);
                    }, () => {
                        this.showLoader = false;
                        this.wrongCredentialVissible = true;
                    });
            });
        }
    }

    saveChanges(gitForm) {
        this.testCredentials(gitForm, () => {
            let tool: ITool = {
                id: this.gitSource.id,
                type: gitForm.controls.type.value,
                username: gitForm.controls.username.value,
                password: gitForm.controls.password.value,
                url: gitForm.controls.url.value
            };

            this.toolService.updateToolAsync(tool).subscribe(
                success => {
                    this.created.emit(success);
                    this.notificationService.showNotification.emit({message: `Git repository: ${tool.url} was successfully updated.`});
                }
            );
        });
    }

    disconnectAccount() {
        this.modalService.showModal('Disconnecting Git', 'You are sure you want to disconnect git connection?').subscribe(result => {
            if (result) {
                this.toolService.deleteToolAsync(this.gitSource.id).subscribe(
                    () => {
                        this.deleted.emit({});
                        this.notificationService.showNotification.emit(
                            {message: `Git repository: ${this.gitSource.url} was successfully disconnected.`});
                    }
                );
            }
        });
    }

    testCredentials(gitForm, onSuccess: () => any) {
        this.wrongCredentialVissible = false;

        let tool: ITool;
        tool = {
            type: gitForm.controls.type.value,
            username: gitForm.controls.username.value,
            password: gitForm.controls.password.value,
            url: gitForm.controls.url.value
        };

        this.toolService.testCredentialsAsync(tool).subscribe(
            () => {
                onSuccess();
            }, () => {
                this.wrongCredentialVissible = true;
            });
    }
}
