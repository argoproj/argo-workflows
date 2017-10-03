import { Component, OnInit } from '@angular/core';
import { FormGroup, FormArray, FormControl, Validators } from '@angular/forms';
import { ActivatedRoute } from '@angular/router';

import { SystemService, NotificationService, ToolService, ModalService, AuthenticationService } from '../../../services';
import { CertificateTool } from '../../../model';
import { LayoutSettings, HasLayoutSettings } from '../../layout/layout.component';
import { CustomRegex } from '../../../common';
import { ToasterService } from 'angular2-toaster/angular2-toaster';

@Component({
    selector: 'ax-system-settings',
    templateUrl: './system-settings.html',
    styles: [ require('./system-settings.scss') ]
})
export class SystemSettingsComponent implements OnInit, HasLayoutSettings, LayoutSettings {
    public systemSettingsEditSubmitted: boolean;
    public certEditSubmitted: boolean;
    public loginSubmitted: boolean;
    public encryptionKey: string;
    public needRelogin: boolean = false;
    public samlLogin: boolean = false;
    public samlButtonLabel: string = '';
    public isSuperAdmin: boolean = false;

    private accessSettingsForm: FormGroup;
    private systemSettingsEditForm: FormGroup;
    private certEditForm: FormGroup;
    private dnsName: string;
    private cert: CertificateTool = new CertificateTool();
    private isEdit: boolean = false;
    private category: string = 'certificate';

    constructor(
            private systemService: SystemService,
            private notificationService: NotificationService,
            private toolService: ToolService,
            private modalService: ModalService,
            private activatedRoute: ActivatedRoute,
            private authenticationService: AuthenticationService,
            private toasterService: ToasterService) {
    }

    async ngOnInit() {
        this.authenticationService.getCurrentUser().then(user => this.isSuperAdmin = user.isSuperAdmin());
        this.systemService.getDnsName().subscribe(res => {
            this.dnsName = res.dnsname;
            this.systemSettingsEditForm = new FormGroup({
                dnsname: new FormControl(this.dnsName, Validators.required),
            });
        });
        this.getCertificate();

        let settings = await this.systemService.getAccessSettings();
        this.accessSettingsForm = new FormGroup({
            trustedCidrs: new FormArray(settings.trusted_cidrs.map(cidr => this.initCidrCtrl(cidr))),
        });

        let res = await this.authenticationService.getAuthSchemas().toPromise();
        let schemas = res.data || [];
        let samlSchema = schemas.find(item => item.name === 'saml');
        if (samlSchema) {
            this.samlLogin = true;
            this.samlButtonLabel = samlSchema.button_label || 'Single Sign On';
        }
    }

    async saveAccessSettings() {
        if (this.accessSettingsForm.valid) {
            await this.systemService.updateAccessSettings({
                trusted_cidrs: this.accessSettingsForm.value.trustedCidrs
            });
            this.notificationService.showNotification.emit({ message: `Access settings was successfully updated.` });
        }
    }

    initCidrCtrl(cidr: string) {
        return new FormControl(cidr, Validators.pattern(CustomRegex.cidr));
    }

    getCidrCtrls(): FormArray {
        return <FormArray> this.accessSettingsForm.controls['trustedCidrs'];
    }

    addCidr() {
        let cidrCtrls = this.getCidrCtrls();
        cidrCtrls.push(this.initCidrCtrl(''));
        this.accessSettingsForm.markAsDirty();
    }

    removeCidr(index: number) {
        let cidrCtrls = this.getCidrCtrls();
        if (cidrCtrls.length > 1) {
            cidrCtrls.removeAt(index);
        }
    }

    get layoutSettings(): LayoutSettings {
        return this;
    }

    get pageTitle(): string {
        return 'System Settings';
    };

    public breadcrumb: { title: string, routerLink?: any[] }[] = [{
        title: `Settings`,
        routerLink: [`/app/settings/overview`],
    }, {
        title: `System Settings`,
    }];

    /**
     * Do save/edit action for system settings
     */
    saveSystemSettingsConfig(form) {
        this.systemSettingsEditSubmitted = true;
        if (form.valid) {
            this.systemService.updateDnsName({dnsname: form.value.dnsname}).subscribe(success => {
                this.notificationService.showNotification.emit(
                    { message: `System settings were successfully updated.` });
            });
        }
    }

    saveCert(form) {
        this.certEditSubmitted = true;
        if (form.valid) {
            if (this.isEdit) {
                this.modalService.showModal(`Update certificate`,
                    `The backend service will be restarted shortly to reflect the certificate change.
                Are you sure you want to update certificate?`)
                    .subscribe(result => {
                        if (result) {
                            this.toolService.updateToolAsync(form.value).subscribe(success => {
                                this.notificationService.showNotification.emit(
                                    {message: `Certificate was successfully updated.`});
                            });
                        }
                    });
            } else {
                this.modalService.showModal(`Create certificate`,
                    `The backend service will be restarted shortly to reflect the certificate change.
                    Are you sure you want to create certificate?`)
                    .subscribe(result => {
                        if (result) {
                            this.toolService.createCertificateTool(form.value).subscribe(success => {
                                this.notificationService.showNotification.emit(
                                    { message: `Certificate was successfully created.` });
                            });
                        }
                    });
            }
        }
    }

    deleteCert(form) {
        this.modalService.showModal(`Delete certificate`,
            `The backend service will be restarted shortly to reflect the certificate change.
                    Are you sure you want to delete certificate?`)
            .subscribe(result => {
                if (result) {
                    this.toolService.deleteToolAsync(this.cert.id).subscribe(success => {
                        this.notificationService.showNotification.emit(
                            { message: `Certificate was successfully deleted.` });
                    });
                }
            });
    }

    getCertificate() {
        this.toolService.getToolsAsync().subscribe(
            success => {
                this.cert = success.data.filter((item: CertificateTool) => {
                    return item.category === this.category;
                })[0];

                this.isEdit = this.cert !== undefined;

                this.certEditForm = new FormGroup({
                    id: new FormControl(this.cert ? this.cert.id : ''),
                    type: new FormControl(this.cert ? this.cert.type : ''),
                    category: new FormControl(this.category)
                });

                // if update 'private_key' and 'public_cert' properties are required
                if (this.isEdit) {
                    this.certEditForm.addControl('private_key',
                        new FormControl(this.cert  && this.cert.hasOwnProperty('private_key') && this.cert.private_key ?
                            this.cert.private_key : '', Validators.required));
                    this.certEditForm.addControl('public_cert',
                        new FormControl(this.cert  && this.cert.hasOwnProperty('public_cert') && this.cert.public_cert ?
                            this.cert.public_cert : '', Validators.required));
                } else {
                    this.certEditForm.addControl('private_key', new FormControl(''));
                    this.certEditForm.addControl('public_cert', new FormControl(''));
                }

                this.certEditForm.updateValueAndValidity();
            }
        );
    }
}
