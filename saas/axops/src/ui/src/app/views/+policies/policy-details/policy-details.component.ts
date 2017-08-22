import { Component, OnInit, OnDestroy } from '@angular/core';
import { ActivatedRoute } from '@angular/router';
import { Subscription } from 'rxjs';

import { PoliciesService, ViewPreferencesService } from '../../../services';
import { Policy, ViewPreferences } from '../../../model';
import { LayoutSettings } from '../../layout';
import { ViewUtils } from '../../../common/view-utils';

@Component({
    selector: 'ax-policy-details',
    templateUrl: './policy-details.html',
    styles: [ require('./policy-details.scss') ],
})
export class PolicyDetailsComponent implements OnInit, OnDestroy, LayoutSettings {
    public parameters: {key: string, value: string}[] = [];

    private policy: Policy;
    private subscriptions: Subscription[] = [];
    private viewPreferences: ViewPreferences;

    constructor(
        private activatedRoute: ActivatedRoute,
        private policiesService: PoliciesService,
        private viewPreferencesService: ViewPreferencesService,
    ) {}

    public ngOnInit() {
        this.activatedRoute.params.subscribe(params => this.getPolicyEditData(params['policyId']));
        this.subscriptions.push(this.viewPreferencesService.getViewPreferencesObservable().subscribe(v => this.viewPreferences = v));
    };

    public ngOnDestroy() {
        this.subscriptions.forEach(item => item.unsubscribe());
        this.subscriptions = [];
    };

    get pageTitle(): string {
        return this.policy && this.policy.name ? this.policy.name : '';
    };

    get breadcrumb(): { title: string, routerLink: any[] }[] {
        return this.policy ? ViewUtils.getBranchBreadcrumb(this.policy.repo, this.policy.branch, '/app/policies/overview', this.viewPreferences, this.policy.name) : null;
    }

    public branchNavPanelUrl = '/app/policies/overview';

    getPolicyEditData(id) {
        this.policiesService.getPolicyById(id).toPromise().then(results => {
            this.policy = results;

            this.parameters = ViewUtils.mapToKeyValue(this.policy.parameters);
        });
    }

    ifOnCron(onEventType: string): boolean {
        return onEventType === 'on_cron';
    }

    onChangeStatus(e: any) {
        let isEnabled = e.target.checked;

        if (isEnabled) {
            this.policiesService.enablePolicy(this.policy.id).subscribe(null, () => this.policy.enabled = !isEnabled);
        } else {
            this.policiesService.disablePolicy(this.policy.id).subscribe(null, () => this.policy.enabled = !isEnabled);
        }
    }
}
