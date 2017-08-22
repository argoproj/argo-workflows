import { Component, Input } from '@angular/core';
import { HumanizedRetentionPolicy } from '../artifact-retention-policy.component';
import { FormGroup } from '@angular/forms';

@Component({
    selector: 'ax-retention-policy-row',
    templateUrl: './retention-policy-row.html',
    styles: [ require('./retention-policy-row.scss') ],
})
export class RetentionPolicyRowComponent {
    @Input()
    public formGroup: FormGroup;

    @Input()
    policyName: string;

    @Input()
    policy: HumanizedRetentionPolicy;

    @Input()
    public name: string = '';

    @Input()
    public editMode: boolean = false;

    @Input()
    public noBorderBottom: boolean = false;

    public getPolicyName(): string {
        return this.policy.name;
    };
}
