import { Component, Input, Output, EventEmitter } from '@angular/core';
import { FormGroup, FormControl, Validators } from '@angular/forms';

import { CustomRegex } from '../../../common/customValidators/CustomRegex';
import { Tool, ITool, REGISTRY_TYPES } from '../../../model';
import { ToolService, ModalService } from '../../../services';
import { NotificationsService } from 'argo-ui-lib/src/components';

@Component({
    selector: 'ax-registry-panel',
    templateUrl: './registry-panel.html',
})
export class RegistryPanelComponent {
    dataSource: ITool = new Tool();
    @Output()
    created: EventEmitter<Tool> = new EventEmitter<Tool>();
    @Output()
    deleted: EventEmitter<any> = new EventEmitter<any>();

    public submitted: boolean;
    public registerTypes = REGISTRY_TYPES;
    private form: FormGroup;
    private isCredentialsConfirmed: boolean = false;
    private wrongCredentialVisible: boolean = false;

    constructor(private toolService: ToolService,
                private notificationsService: NotificationsService,
                private modalService: ModalService) {
        this.form = new FormGroup({
            type: new FormControl(''),
            username: new FormControl('', [Validators.pattern(CustomRegex.myToolsUsername)]),
            password: new FormControl(''),
        });
    }

    @Input()
    set type(val: string) {
        this.form.controls['type'].setValue(val);

        if (val === REGISTRY_TYPES.privateRegistry) {
            this.form.addControl('hostname', new FormControl(this.dataSource.hostname, [Validators.required]));
        } else {
            this.form.removeControl('hostname');
        }
    }

    get type(): string {
        return this.form.controls['type'].value;
    }

    @Input()
    set source(value: Tool) {
        this.dataSource = value || new Tool();

        let params = {
            type: this.type,
            username: this.dataSource.username,
            password: this.dataSource.password,
        };
        if (this.type === REGISTRY_TYPES.privateRegistry) {
            params = Object.assign(params, { hostname: this.dataSource.hostname });
        }
        this.form.reset(params);
    }

    get isAccountConnected(): boolean {
        return !!this.dataSource.id;
    }

    connect() {
        this.submitted = true;

        if (!this.form.valid) {
            return;
        }

        this.toolService.postContainerRegistry(this.form.value).subscribe(
            success => {
                this.dataSource = success;
                this.notificationsService.success(`Container registry was successfully created.`);
                this.created.emit(success);
            }, () => {
                this.isCredentialsConfirmed = false;
                this.wrongCredentialVisible = true;
            }
        );
    }

    update() {
        this.submitted = true;

        if (!this.form.valid) {
            return;
        }

        let data = Object.assign({}, this.dataSource, this.form.value);

        this.toolService.updateToolAsync(data).subscribe(success => {
            this.dataSource = success;
            this.notificationsService.success(`Container registry was successfully updated.`);
        });
    }

    testCredentials() {
        let data = this.form.value;

        this.toolService.testCredentialsAsync(data).subscribe(() => {
            this.isCredentialsConfirmed = true;
            this.wrongCredentialVisible = false;
        }, () => {
            this.isCredentialsConfirmed = false;
            this.wrongCredentialVisible = true;
        });
    }

    disconnect() {
        this.modalService.showModal('Disconnecting registry', 'You are sure you want to disconnect registry connection?')
            .subscribe(result => {
                if (!result) {
                    return;
                }

                this.toolService.deleteToolAsync(this.dataSource.id).subscribe(() => {
                    this.dataSource = new Tool();
                    (<FormControl>this.form.controls['username']).setValue('');
                    (<FormControl>this.form.controls['password']).setValue('');

                    if (this.type === REGISTRY_TYPES.privateRegistry) {
                        (<FormControl>this.form.controls['hostname']).setValue('');
                    }

                    this.notificationsService.success(`Registry was successfully disconnected.`);
                    this.deleted.emit({});
                });
            });
    }
}
