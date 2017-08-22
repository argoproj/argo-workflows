import { Component, OnDestroy, OnInit, ViewChild } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { Subscription } from 'rxjs';

import { DropDownComponent } from 'argo-ui-lib/src/components';

import { LayoutSettings } from '../../layout/layout.component';
import { PoliciesService, NotificationService } from '../../../services';
import { Policy, Rule } from '../../../model';

import { NotificationCreationPanelComponent } from './notification-creation-panel/notification-creation-panel.component';

export class PolicyNotification {
    id: string;
    name: string;
    criteria: string[];
    recipients: string[];
    enabled: boolean;
}

@Component({
    selector: 'ax-notification-management',
    templateUrl: './notification-management.html',
    styles: [require('./notification-management.scss')],
})
export class NotificationManagementComponent implements LayoutSettings, OnInit, OnDestroy {
    public configuredRules: Rule[] = [];
    public enabledPolicies: PolicyNotification[] = [];
    public loadNotification: boolean = false;
    public loadPolicy: boolean = false;
    public isNotificationCriteriaPanelVisible: boolean = false;

    @ViewChild('rulesDropdown')
    public rulesDropdown: DropDownComponent;

    @ViewChild(NotificationCreationPanelComponent)
    private notificationCreationPanelComponent: NotificationCreationPanelComponent;

    private subscriptions: Subscription[] = [];
    private updateRuleId: string;

    constructor(private activatedRoute: ActivatedRoute,
                private router: Router,
                private policiesService: PoliciesService,
                private notificationService: NotificationService) {
    }

    public ngOnInit() {
        this.activatedRoute.params.subscribe(params => {
            this.updateRuleId = params['ruleId'] || null;
            this.toolbarFilters.model = params['rules'] && decodeURIComponent(params[ 'rules' ]).split(',') || [];

            if (this.updateRuleId) {
                this.openNotificationCreationPanel();
                // JSON.parse(JSON.stringify())) deep copy
                this.notificationCreationPanelComponent.ruleToEdit =
                    this.configuredRules.length ? JSON.parse(JSON.stringify(this.configuredRules.filter(i => i.rule_id === this.updateRuleId)[0])) : null;
            }
        });

        this.getNotification();
        this.getPolicies();
        this.getChannels();
    }

    public ngOnDestroy() {
        this.subscriptions.forEach(subscription => subscription.unsubscribe());
    }

    get layoutSettings(): LayoutSettings {
        return this;
    }

    // set title
    get pageTitle(): string {
        return 'Notification Rules';
    };

    // plus button action
    public globalAddAction() {
        this.openNotificationCreationPanel();
    }

    public toolbarFilters = {
        data: [],
        model: [],
        onChange: () => {
                this.router.navigate([`/app/settings/notification-management`, this.getRouteParams()]);
        }
    };

    public breadcrumb: { title: string, routerLink?: any[] }[] = [{
        title: `Settings`,
        routerLink: [`/app/settings/overview`],
    }, {
        title: `Notification Rules`,
    }];

    public openNotificationCreationPanel() {
        this.isNotificationCriteriaPanelVisible = true;
    }

    public closeNotificationCreationPanel() {
        // reset updated item id
        this.updateRuleId = null;
        this.isNotificationCriteriaPanelVisible = false;
        this.router.navigate([`/app/settings/notification-management`, this.getRouteParams()]);
    }

    public openEditNotificationPanel(ruleId: string) {
        this.updateRuleId = ruleId;
        this.router.navigate([`/app/settings/notification-management`, this.getRouteParams()]);
    }

    public moveToPolicyDetails(policyId: string): void {
        this.router.navigate([`/app/policies/details/${policyId}`]);
    }

    public reloadRules() {
        this.getNotification();
    }

    private async getNotification() {
        try {
            this.configuredRules = [];
            this.loadNotification = true;
            this.configuredRules = await this.notificationService.getRules();
            this.loadNotification = false;

            // we don't support get rule by id. Once we get list of rules and we have ruleId in query string, open the panel and display selected rule datd
            if (this.updateRuleId) {
                this.openNotificationCreationPanel();
                this.notificationCreationPanelComponent.ruleToEdit = this.configuredRules.filter(i => i.rule_id === this.updateRuleId)[0];
            }
        } catch (err) {
            this.configuredRules = [];
            this.loadNotification = false;
        }
    }

    private async getChannels() {
        let channels = await this.notificationService.getChannels();
        this.toolbarFilters.data = channels.map(item => { return {name: item, value: item}; });
    }

    private getPolicies() {
        this.loadPolicy = true;
        this.subscriptions.push(this.policiesService.getPolicies({
            enabled: true,
        }, true).subscribe(result => {
            this.loadPolicy = false;
            this.enabledPolicies = result.data.filter(i => i.hasOwnProperty('notifications')).map((item: Policy) => {
                let notifications =
                    item.hasOwnProperty('notifications') && item.notifications.length && item.notifications[0] ? item.notifications[0] : {};

                return {
                    id: item.id,
                    name: item.name,
                    criteria: notifications.hasOwnProperty('when') ? notifications['when'] : [],
                    recipients: notifications.hasOwnProperty('whom') ? notifications['whom'] : [],
                    enabled: true
                };
            });
        }, error => {
            this.loadPolicy = false;
            this.enabledPolicies = [];
        }));
    }

    private getRouteParams() {
        let params = {};

        if (this.updateRuleId) {
            params['ruleId'] = this.updateRuleId;
        }

        if (this.toolbarFilters.model) {
            params['rules'] =  encodeURIComponent(this.toolbarFilters.model.join(','));
        }

        return params;
    }
}
