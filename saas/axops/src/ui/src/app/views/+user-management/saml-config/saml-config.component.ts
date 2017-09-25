import * as _ from 'lodash';
import { Component, OnInit } from '@angular/core';
import { FormGroup, FormControl, Validators, ValidatorFn } from '@angular/forms';
import { ToolService, NotificationService, ModalService } from '../../../services';
import { SamlConfigTool, SamlInfo } from '../../../model';

@Component({
    selector: 'ax-saml-config',
    templateUrl: './saml-config.html',
    styles: [ require('./saml-config.scss') ],
})
export class SamlConfigComponent implements OnInit {
    public submitted: boolean;

    private samlConfig: SamlConfigTool = new SamlConfigTool();
    private samlConfigEditForm: FormGroup;
    private isEdit: boolean = false;
    private samlInfo: SamlInfo = null;
    private hideAdvanced: boolean = true;

    static publicCertRequired(propertyFirst: string, propertySecond: string): ValidatorFn {
        return (group: FormGroup): {[key: string]: any} => {
            if (group.controls['idp_public_cert'] && (group.value[propertyFirst] || group.value[propertySecond])
                && Validators.required(group.controls['idp_public_cert'])) {
                return {idp_public_cert_required: true};
            } else {
                return null;
            }
        };
    }

    constructor(private toolsService: ToolService,
        private notificationService: NotificationService,
        private modalService: ModalService) {
    }

    public ngOnInit() {
        Promise.all([
            this.toolsService.getSAMLInfo().toPromise(),
            this.toolsService.getToolsAsync({category: 'authentication'}, true).toPromise()
                .then(res => res.data)]).then(([samlInfo, samlConfigs]: [SamlInfo, SamlConfigTool[]]) => {

            this.samlInfo = samlInfo;
            if (samlConfigs.length > 0) {
                this.samlConfig = samlConfigs[0];
                this.isEdit = true;
            }
            this.initForm();
        });
    }

    // Toggle Advanced setting in UI
    toggleAdvanced() {
        this.hideAdvanced = !this.hideAdvanced;
    }

    initForm() {
        this.samlConfigEditForm = new FormGroup({
            button_label: new FormControl(this.samlConfig.button_label, Validators.required),
            email_attribute: new FormControl(this.samlConfig.email_attribute, [Validators.required]),
            first_name_attribute: new FormControl(this.samlConfig.first_name_attribute, Validators.required),
            group_attribute: new FormControl(this.samlConfig.group_attribute, Validators.required),
            idp_public_cert: new FormControl(this.samlConfig.idp_public_cert),
            idp_sso_url: new FormControl(this.samlConfig.idp_sso_url, Validators.required),
            last_name_attribute: new FormControl(this.samlConfig.last_name_attribute, Validators.required),
            sp_description: new FormControl(this.samlConfig.sp_description, Validators.required),
            sp_display_name: new FormControl(this.samlConfig.sp_display_name, Validators.required),
            deflate_response_encoded: new FormControl(this.samlConfig.deflate_response_encoded),
            sign_request: new FormControl(this.samlConfig.sign_request),
            signed_response: new FormControl(this.samlConfig.signed_response),
            signed_response_assertion: new FormControl(this.samlConfig.signed_response_assertion)
        });
        // TODO Fix for validator: publicCertRequired. It stopped work after change to A2 Finall
        // }, null, SamlConfigComponent.publicCertRequired('signed_response', 'signed_response_assertion'));
    }

    /**
     * Do save/edit action for SAML config
     */
    saveSamlConfig(samlConfigEditForm) {
        this.submitted = true;
        if (samlConfigEditForm.valid) {
            this.samlConfig = _.extend(this.samlConfig, this.samlConfigEditForm.value);

            if (this.isEdit) {
                this.toolsService.updateToolAsync(this.samlConfig).subscribe(success => {
                    this.notificationService.showNotification.emit(
                        { message: `SSO configuration was successfully updated.` });
                });
            } else {
                this.toolsService.createSamlConfigTool(this.samlConfig).subscribe(success => {
                    this.notificationService.showNotification.emit(
                        { message: `SSO configuration was successfully created.` });
                    this.samlConfig.id = success.id;
                    this.isEdit = true;
                });
            }
        }
    }
    /**
     * An admin can now remove SSO setup from the application
     */
    removeSSO() {
        this.modalService.showModal('Remove Single Sign-On',
            'Are you sure you want remove SSO setup from your account? This action cannot be reverted.').subscribe(result => {
            if (result) {
                this.toolsService.deleteToolAsync(this.samlConfig.id).subscribe(
                    res => {
                        this.notificationService.showNotification.emit({ message: `Single Sign-On is disabled.` });

                        // reload the app
                        window.location.reload();
                    }
                );

            }
        });
    }
}
