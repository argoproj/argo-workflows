import { Component, Output, EventEmitter, OnInit } from '@angular/core';

import { PolicyNotification } from '../../../model';

@Component({
    selector: 'ax-setup-job-notifications',
    templateUrl: './setup-job-notifications.html',
    styles: [ require('./setup-job-notifications.scss') ],
})
export class SetupJobNotificationsComponent implements OnInit {
    @Output()
    public onChange: EventEmitter<any> = new EventEmitter();

    public notificationRules: PolicyNotification[] = [];

    public eventTypes: any = {
        items: [
            {name: 'on_change', value: 'on_change', checked: false},
            {name: 'on_cron', value: 'on_cron', checked: false},
            {name: 'on_failure', value: 'on_failure', checked: false},
            {name: 'on_pull_request', value: 'on_pull_request', checked: false},
            {name: 'on_pull_request_merge', value: 'on_pull_request_merge', checked: false},
            {name: 'on_push', value: 'on_push', checked: false},
            {name: 'on_start', value: 'on_start', checked: false},
            {name: 'on_success', value: 'on_success', checked: false},
        ],
        messages: {
            name: 'JOB EVENTS',
        },
        isVisible: false,
        isStaticList: true,
        isDisplayedInline: true,
        isArgoUsersAndGroupsVisible: false
    };

    public eventTypesList: any[] = [];
    public isVisibleUserSelectorPanel: boolean = false;
    public axUsersAndGroupsList: string[] = [];
    public selectedId: number = 0;


    ngOnInit() {
        if (!this.notificationRules.length) {
            this.addNotificationRule();
        }
    }

    public onEventTypeChange(when: string[], index) {
        console.log('onEventTypeChange', event, index);
        this.notificationRules[index].when = when;
        console.log('this.notificationRules', this.notificationRules);
    }

    public addNotificationRule() {
        this.eventTypesList.push(JSON.parse(JSON.stringify(this.eventTypes)));
        this.notificationRules.push({ whom: [], when: []});
    }

    public removeNotificationRule(index) {
        this.notificationRules.splice(index, 1);
        this.eventTypesList.splice(index, 1);
    }

    public openUserSelectorPanel(index) {
        this.isVisibleUserSelectorPanel = true;
        this.selectedId = index;
    }

    public closeUserSelectorPanel() {
        this.isVisibleUserSelectorPanel = false;
    }

    public updateUsersList(whom: string[]) {
        console.log('updateUsersList', whom, this.selectedId, this. notificationRules);
        this.notificationRules[this.selectedId].whom = whom;
    }
}
