import { Component, OnInit, QueryList, ViewChildren } from '@angular/core';
import { FormGroup, Validators, FormControl } from '@angular/forms';

import { LayoutSettings, HasLayoutSettings } from '../../layout/layout.component';
import { RetentionPolicyRowComponent } from './retention-policy-row/retention-policy-row.component';
import { NotificationsService } from 'argo-ui-lib/src/components';
import { RetentionPolicyService } from '../../../services';
import { ArtifactsService, ModalService } from '../../../services';
import { ARTIFACT_TYPES, RetentionPolicy } from '../../../model';
import { TimeOperations } from '../../../common/timeOperations/timeOperations';
import { CustomValidators } from '../../../common/customValidators/CustomValidators';

export class HumanizedRetentionPolicy extends RetentionPolicy {
    private _days: number;
    private _unit: string;

    get days(): number {
        return this._days;
    }

    get unit(): string {
        return this._unit;
    }

    set days(value: number) {
        this._days = value;
        this._unit = this.days > 1 ? 'days' : 'day';
        this.policy = TimeOperations.unitInMilliseconds(value, 'day');
    }

    constructor(data?) {
        super(data);
        if (typeof data === 'object' && data.hasOwnProperty('policy')) {
            this._days = TimeOperations.millisecondsAsDays(this.policy);
            this._unit = this.days > 1 ? 'days' : 'day';
        }
    }
}

export class HumanizedRetentionPolicies {
    totalNumber: number = 0;
    totalSize: number = 0;
    'ax-log': HumanizedRetentionPolicy;
    'user-log': HumanizedRetentionPolicy;
    'internal': HumanizedRetentionPolicy;
    'exported': HumanizedRetentionPolicy;
}

@Component({
    selector: 'ax-artifact-retention-policy',
    templateUrl: './artifact-retention-policy.html',
    styles: [ require('./artifact-retention-policy.scss') ],
})
export class ArtifactRetentionPolicyComponent implements LayoutSettings, HasLayoutSettings, OnInit {
    public policyForm: FormGroup;

    @ViewChildren(RetentionPolicyRowComponent)
    public retentionPolicyRows: QueryList<RetentionPolicyRowComponent>;

    public retentionTypes = ARTIFACT_TYPES;
    public editMode: boolean = false;
    public dataLoaded: boolean = false;
    private policies: HumanizedRetentionPolicies = new HumanizedRetentionPolicies();

    // Layout settings
    public pageTitle: string = 'Artifact Retention Policy';
    public breadcrumb: { title: string, routerLink?: any[] }[] = [{
        title: `Settings`,
        routerLink: [`/app/settings/overview`],
    }, {
        title: `Artifact Retention Policy`,
    }];

    constructor(private notificationsService: NotificationsService,
                private modalService: ModalService,
                private artifactsService: ArtifactsService,
                private retentionPolicyService: RetentionPolicyService) {
    }

    ngOnInit() {
        this.getRetentionPolicies();
    }

    get layoutSettings(): LayoutSettings {
        return this;
    }

    public onSubmit(model: FormGroup): void {
        this.editMode = false;
        this.retentionPolicyRows.forEach((row: RetentionPolicyRowComponent) => {
            let policyName = row.getPolicyName();
            if (model.controls.hasOwnProperty(policyName)) {
                let control: FormControl = <FormControl>model.get(policyName).get('policy');
                if (control.dirty) {
                    let policyValue = control.value;
                    this.policies[policyName].days = policyValue;
                    this.retentionPolicyService.updateRetentionPolicy(
                        policyName,
                        this.policies[policyName].policy,
                        true
                    ).subscribe((success: any[]) => {
                        control.reset(policyValue);
                    }, error => {
                        this.notificationsService.internalError();
                    });
                }
            }
        });
    }

    public onClose(): void {
        this.editMode = false;
        this.retentionPolicyRows.forEach((row: RetentionPolicyRowComponent) => {
            let policyName = row.getPolicyName();
            let control: FormControl = <FormControl>row.formGroup.get('policy');
            control.reset(this.policies[policyName].days);
        });
    }

    public onEdit(): void {
        this.editMode = true;
    }

    public cleanArtifacts(): void {
        this.modalService.showModal('Reclaim Space Now',
            `Are you sure you want to clean your artifacts according to retention policies to reclaim space?`)
            .subscribe(result => {
                if (result) {
                    this.artifactsService.cleanArtifacts(true).subscribe(() => {
                        this.notificationsService.success(`Process to reclaim space started.`);
                        this.getRetentionPolicies();
                    }, () => {
                        this.notificationsService.error(`Unable to start reclaimation process.`);
                    });
                }
            });
    }

    private getRetentionPolicies(): void {
        this.retentionPolicyService.getRetentionPolicies(true).subscribe((success: any) => {
            let group: any = {};
            this.policies.totalSize = 0;
            this.policies.totalNumber = 0;
            success.data.forEach((policy: RetentionPolicy) => {
                this.policies[policy.name] = new HumanizedRetentionPolicy(policy);
                this.policies.totalSize += policy.total_size;
                this.policies.totalNumber += policy.total_number;
                group[policy.name] = new FormGroup({
                    policy: new FormControl(this.policies[policy.name].days,
                        [Validators.required, CustomValidators.number({min: 1})]),
                });
            });
            this.policyForm = new FormGroup(group);
            this.dataLoaded = true;
        }, error => {
            this.notificationsService.internalError();
        });
    }
}
