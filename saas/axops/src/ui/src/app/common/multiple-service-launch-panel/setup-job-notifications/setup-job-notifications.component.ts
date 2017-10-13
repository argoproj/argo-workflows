import * as _ from 'lodash';
import { Component, OnInit } from '@angular/core';

import { FilterMultiSelect } from 'argo-ui-lib/src/components';

import { PolicyNotification } from '../../../model';
import { CustomRegex } from '../../customValidators/CustomRegex';
import { AuthenticationService } from '../../../services';

interface NotificationObject {
    jobEvents: FilterMultiSelect;
    isArgoUsersAndGroupsVisible: boolean;
    isSlackChannelsVisible: boolean;
    isEmailVisible: boolean;
    rules: PolicyNotification;
    outsideUsers: string[];
    filteredOutsideUsers: string[];
    validationMessages: any;
}

@Component({
    selector: 'ax-setup-job-notifications',
    templateUrl: './setup-job-notifications.html',
    styles: [ require('./setup-job-notifications.scss') ],
})
export class SetupJobNotificationsComponent implements OnInit {

    public notification: NotificationObject = {
        jobEvents: {
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
            selectedItems: [],
            messages: {
                name: 'JOB EVENTS',
            },
            isVisible: false,
            isStaticList: true,
            isDisplayedInline: true,
        },

        isArgoUsersAndGroupsVisible: false,
        isSlackChannelsVisible: false,
        isEmailVisible: false,
        rules: {
            whom: [],
            when: []
        },
        outsideUsers: [],
        filteredOutsideUsers: [],
        validationMessages: {
            jobEvents: { show: false, text: 'You have to choose at least one Job Event' },
            wrongFormatRecipients: { show: false, text: 'Recipients have to be an email format' },
            missingRecipients: { show: false, text: 'You have to choose at least one recipient from any of the groups' },
        }
    };

    public rules: PolicyNotification[] = [];
    public notificationsList: any[] = [];
    public isVisibleUserSelectorPanel: boolean = false;
    public isVisibleSlackPanel: boolean = false;
    public selectedId: number = 0;

    constructor(private authenticationService: AuthenticationService) {
    }

    ngOnInit() {
        if (!this.rules.length) {
            this.addDefaultNotificationRule();
        }
    }

    public onRuleChange(when: string[], index) {
        this.notificationsList[index].rules.when = when;
        this.notificationsList[index].validationMessages.jobEvents.show = false;
    }

    public addNotificationRule() {
        this.notificationsList.push(JSON.parse(JSON.stringify(this.notification)));
    }

    public removeNotificationRule(index) {
        this.notificationsList.splice(index, 1);
    }

    public openUserSelectorPanel(index) {
        this.isVisibleUserSelectorPanel = true;
        this.selectedId = index;
        this.notificationsList[index].validationMessages.missingRecipients.show = false;
    }

    public closeUserSelectorPanel() {
        this.isVisibleUserSelectorPanel = false;
    }

    public openSlackChannelPanel(index) {
        this.isVisibleSlackPanel = true;
        this.selectedId = index;
        this.notificationsList[index].validationMessages.missingRecipients.show = false;
    }

    public closeSlackChannelPalen() {
        this.isVisibleSlackPanel = false;
    }

    public getOutsideUsers(index) {
        this.notificationsList[index].filteredOutsideUsers =
            this.notificationsList[index].rules.whom.recipients.filter(recipient => recipient.indexOf('@user') !== -1).sort();
        this.notificationsList[index].outsideUsers = this.notificationsList[index].filteredOutsideUsers.map(user => user.substring(0, user.indexOf('@user')));
        return this.notificationsList[index].rules.whom.filter(recipient => recipient.indexOf('@user') !== -1).sort();
    }

    public getOnlyUsersAndGroups(index) {
        return this.notificationsList.length ?
            this.notificationsList[index].rules.whom.filter(recipient => recipient.indexOf('@slack') === -1 && recipient.indexOf('@user') === -1).sort() : [];
    }

    public getOnlySlackChannels(index) {
        return this.notificationsList.length ?
            this.notificationsList[index].rules.whom.filter(recipient => recipient.indexOf('@slack') !== -1).sort() : [];
    }

    public updateOnlyUsersAndGroupsList(users: string[]) {
        this.notificationsList[this.selectedId].rules.whom =
            this.notificationsList[this.selectedId].rules.whom.filter(recipient => (recipient.indexOf('@slack') !== -1 || recipient.indexOf('@user') !== -1)).sort();
        this.updateNotificationWhomList(users);
    }

    public updateSlackChannelsList(channels: string[]) {
        this.notificationsList[this.selectedId].rules.whom =
            this.notificationsList[this.selectedId].rules.whom.filter(recipient => (recipient.indexOf('@slack') === -1)).sort();
        let axSlackChannelsList = channels.map(channel => `${channel}@slack`);
        this.updateNotificationWhomList(axSlackChannelsList);
    }

    public updateNotificationWhomList(list: string[]) {
        this.notificationsList[this.selectedId].rules.whom =
            this.notificationsList[this.selectedId].rules.whom.concat(list).filter((value, index, self) => self.indexOf(value) === index );
    }

    public updateOutsideUsers(users: string, index) {
        this.selectedId = index;
        this.notificationsList[index].outsideUsers = users.split(',');
        this.notificationsList[index].filteredOutsideUsers =
            this.notificationsList[index].outsideUsers.filter(user => CustomRegex.emailPattern.test(user.trim())).map(user => `${user}@user`);
        this.notificationsList[index].validationMessages.wrongFormatRecipients.show = false;
        this.notificationsList[index].validationMessages.missingRecipients.show = false;
    }

    public argoUsersAndGroupsCheckboxChange(notification, index) {
        notification.isArgoUsersAndGroupsVisible = !notification.isArgoUsersAndGroupsVisible;
        if (!notification.isArgoUsersAndGroupsVisible) {
            notification.rules.whom = notification.rules.whom.filter(recipient => (recipient.indexOf('@slack') !== -1 || recipient.indexOf('@user') !== -1)).sort();
        } else {
            this.openUserSelectorPanel(index);
        }
    }

    public emailCheckboxChange(notification) {
        notification.isEmailVisible = !notification.isEmailVisible;
        if (!notification.isEmailVisible) {
            notification.outsideUsers = [];
            notification.filteredOutsideUsers = [];
            notification.rules.whom = notification.rules.whom.filter(recipient => (recipient.indexOf('@user') === -1)).sort();
        }
    }

    public slackChannelsCheckboxChange(notification, index) {
        notification.isSlackChannelsVisible = !notification.isSlackChannelsVisible;
        if (!notification.isSlackChannelsVisible) {
            notification.rules.whom = notification.rules.whom.filter(recipient => (recipient.indexOf('@slack') === -1)).sort();
        } else {
            this.openSlackChannelPanel(index);
        }
    }

    public getNotifications(): PolicyNotification[] {
        let isAnyError = false;
        this.notificationsList.forEach(notification => {
            if (!notification.rules.when.length) {
                notification.validationMessages.jobEvents.show = true;
                isAnyError = true;
            }

            if (notification.outsideUsers.length !== notification.filteredOutsideUsers.length && notification.outsideUsers.toString().length) {
                notification.validationMessages.wrongFormatRecipients.show = true;
                isAnyError = true;
            } else {
                // remove all '@user' elements from rules.whom
                notification.rules.whom = notification.rules.whom.filter(item => item.indexOf('@user') === -1);
                this.updateNotificationWhomList(notification.filteredOutsideUsers);
            }

            if (!notification.rules.whom.length) {
                notification.validationMessages.missingRecipients.show = true;
                isAnyError = true;
            }
        });

        // if there is error in any notification rule, don't allow to submit
        if (isAnyError) {
            return;
        }

        this.rules = this.notificationsList.map(notification => {
            return notification.rules;
        });

        return _.uniqBy(this.rules, v => [v.whom, v.when].join());
    }

    private addDefaultNotificationRule() {
        let notification = JSON.parse(JSON.stringify(this.notification));
        notification.rules = {
            whom: [this.authenticationService.getUsername()],
            when: ['on_failure', 'on_success']
        };
        notification.isArgoUsersAndGroupsVisible = true;

        this.notificationsList.push(notification);
    }
}
