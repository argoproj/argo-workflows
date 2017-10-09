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
        isArgoUsersAndGroupsVisible: false,
        isSlackChannelsVisible: false,
        notificationRules: {
            whom: [],
            when: []
        }
    };

    public notificationRules: PolicyNotification[] = [];
    public eventTypesList: any[] = [];
    public isVisibleUserSelectorPanel: boolean = false;
    public isVisibleSlackPanel: boolean = false;
    public axUsersAndGroupsList: string[] = [];
    public axSlackChannelsList: string[] = [];
    public selectedId: number = 0;


    ngOnInit() {
        if (!this.notificationRules.length) {
            this.addNotificationRule();
        }
    }

    public onEventTypeChange(when: string[], index) {
        this.eventTypes[index].notificationRules.when = when;
    }

    public addNotificationRule() {
        this.eventTypesList.push(JSON.parse(JSON.stringify(this.eventTypes)));
    }

    public removeNotificationRule(index) {
        this.eventTypesList.splice(index, 1);
    }

    public openUserSelectorPanel(index) {
        this.isVisibleUserSelectorPanel = true;
        this.selectedId = index;
    }

    public closeUserSelectorPanel() {
        this.isVisibleUserSelectorPanel = false;
    }

    public openSlackChannelPanel(index) {
        this.isVisibleSlackPanel = true;
        this.selectedId = index;
    }

    public closeSlackChannelPalen() {
        this.isVisibleSlackPanel = false;
    }

    public getOutsideUsers(index) {
        return this.eventTypesList[index].notificationRules.whom.filter(recipient => recipient.indexOf('@user') !== -1).sort();
    }

    public getOnlyUsersAndGroups(index) {
        return this.eventTypesList[index].notificationRules.whom.filter(recipient => recipient.indexOf('@slack') === -1 && recipient.indexOf('@user') === -1).sort();
    }

    public getOnlySlackChannels(index) {
        return this.eventTypesList[index].notificationRules.whom.filter(recipient => recipient.indexOf('@slack') !== -1).sort();
    }

    public updateUsersList(users: string[]) {
        this.updateNotificationWhomList(users);
    }

    public updateSlackChannelsList(channels: string[]) {
        let axSlackChannelsList = channels.map(channel => `${channel}@slack`);
        this.updateNotificationWhomList(axSlackChannelsList);
    }

    public updateNotificationWhomList(list: string[]) {
        this.eventTypesList[this.selectedId].notificationRules.whom =
            this.eventTypesList[this.selectedId].notificationRules.whom.concat(list).filter((value, index, self) => self.indexOf(value) === index );
    }
}
