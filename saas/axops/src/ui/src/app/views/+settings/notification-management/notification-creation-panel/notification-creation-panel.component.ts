import { Component, Input, Output, EventEmitter, OnInit } from '@angular/core';
import { FormGroup, FormControl, Validators } from '@angular/forms';

import { NotificationsService, FilterMultiSelect } from 'argo-ui-lib/src/components';

import { Rule } from '../../../../model';
import { NotificationService, ToolService } from '../../../../services';
import { ViewUtils } from '../../../../../app/common';
import { CustomRegex } from '../../../../common/customValidators/CustomRegex';

export class Criteria {
    selectedEventTypes: string[];
    selectedEventSeverity: string[];
    eventTypes: FilterMultiSelect;
    eventSeverity: FilterMultiSelect;

    constructor(
        selectedEventTypes: string[],
        selectedEventSeverity: string[],
        eventTypes: FilterMultiSelect,
        eventSeverity: FilterMultiSelect) {
        this.selectedEventTypes = selectedEventTypes || [];
        this.selectedEventSeverity = selectedEventSeverity || [];
        this.eventTypes = eventTypes;
        this.eventSeverity = eventSeverity;
    }
}

@Component({
    selector: 'ax-notification-creation-panel',
    templateUrl: './notification-creation-panel.html',
    styles: [require('./notification-creation-panel.scss')]
})
export class NotificationCreationPanelComponent implements OnInit {
    public isVisibleCriteriaLoader: boolean = false;
    public criteriaEvent: Criteria;
    public axUsersAndGroupsList: string[] = [];
    public axSlackChannelsList: string[] = [];
    public outsideUsers: string[] = [];
    public filteredOutsideUsers: string[] = [];
    public notificationCreationForm: FormGroup;
    public rule: Rule;
    public isFirstStep: boolean = true;
    public isVisibleUserSelectorPanel: boolean = false;
    public isVisibleSlackPanel: boolean = false;
    public isSlackIntegrationConfigured: boolean = false;
    public channels: string[] = [];
    public severities: string[] = [];
    public validationMessages: any = {
        ruleName: { show: false, text: 'You have to set Notification Rule Name' },
        eventType: { show: false, text: 'You have to choose at least one Event Type' },
        eventSeverity: { show: false, text: 'You have to choose at least one Event Severity' },
        missingRecipients: { show: false, text: 'You have to choose at least one recipient from any of the groups' },
        wrongFormatRecipients: { show: true, text: 'Recipients have to be an email format' }
    };

    @Input()
    set isVisible(value: boolean) {
        this.isNotificationCriteriaPanelVisible = value;
        if (!this.channels.length && !this.severities.length && !this.isVisibleCriteriaLoader) {
            this.getAndAddCriteria();
        } else {
            this.addCriteria();
        }
    }

    @Input()
    set ruleToEdit(rule: Rule) {
        if (rule) {
            this.rule = rule;
            this.notificationCreationForm.controls['name'].setValue(rule.name);
            this.notificationCreationForm.controls['enabled'].setValue(rule['enabled'] || false);
            this.notificationCreationForm.controls['channels'].setValue(rule['channels'] || []);
            this.notificationCreationForm.controls['severities'].setValue(rule['severities'] || []);
            this.notificationCreationForm.controls['recipients'].setValue(rule['recipients'] || []);
        }
    }

    @Output()
    public onClose: EventEmitter<any> = new EventEmitter<any>();

    @Output()
    public onUpdate: EventEmitter<any> = new EventEmitter<any>();

    private eventTypes: FilterMultiSelect = {
        items: [],
        messages: {
            name: 'Event Type',
        },
        isVisible: false,
        isStaticList: true,
        isDisplayedInline: true
    };
    private eventSeverity: FilterMultiSelect = {
        items: [],
        messages: {
            name: 'Event Severity',
        },
        isVisible: false,
        isStaticList: true,
        isDisplayedInline: true
    };
    private isNotificationCriteriaPanelVisible: boolean = false;

    constructor(private notificationService: NotificationService,
                private notificationsService: NotificationsService,
                private toolService: ToolService) {
    }

    public ngOnInit() {
        this.notificationCreationForm = new FormGroup({
            name: new FormControl('', Validators.required),
            enabled: new FormControl(false),
            channels: new FormControl([], Validators.required),
            severities: new FormControl([], Validators.required),
            recipients: new FormControl([], Validators.required)
        });

        this.notificationCreationForm.valueChanges.subscribe(data => {
            this.clearValidators();
        });
    }

    public next() {
        if (this.notificationCreationForm.controls['name'].invalid ||
            this.notificationCreationForm.controls['channels'].invalid ||
            this.notificationCreationForm.controls['severities'].invalid) {
            this.validate();
            return;
        }

        // check if slack integration is configured
        this.checkIfSlackIntergrationConfigured();
        this.shareUsersToLists();
        this.isFirstStep = false;
    }

    public validate() {
        this.validationMessages.ruleName.show = this.notificationCreationForm.controls['name'].invalid;
        this.validationMessages.eventType.show = this.notificationCreationForm.controls['channels'].invalid;
        this.validationMessages.eventSeverity.show = this.notificationCreationForm.controls['severities'].invalid;
        this.validationMessages.missingRecipients.show = this.notificationCreationForm.controls['recipients'].invalid;
        this.validationMessages.wrongFormatRecipients.show = this.outsideUsers.length !== this.filteredOutsideUsers.length && this.outsideUsers.toString().length;
    }

    public async submit() {
        if (this.notificationCreationForm.invalid || this.outsideUsers.length !== this.filteredOutsideUsers.length && this.outsideUsers.toString().length) {
            this.validate();
            return;
        }

        this.onClose.emit();
        if (this.rule) {
            await this.updateRule();
        } else {
            await this.createRule();
        }
        this.closePanel();
        this.onUpdate.emit();
    }

    public addCriteria() {
        let selectedEventTypes = [];
        let selectedEventSeverity = [];
        if (this.notificationCreationForm.controls['channels'].value) {
            selectedEventTypes = this.notificationCreationForm.controls['channels'].value;
        }
        if (this.notificationCreationForm.controls['severities'].value) {
            selectedEventSeverity = this.notificationCreationForm.controls['severities'].value;
        }

        this.criteriaEvent = new Criteria(selectedEventTypes, selectedEventSeverity, JSON.parse(JSON.stringify(this.eventTypes)), JSON.parse(JSON.stringify(this.eventSeverity)));
    }

    public closePanel() {
        this.onClose.emit();
        // reset rule
        this.ruleToEdit = null;
        // reset form
        this.notificationCreationForm.reset();
        // reset criteria
        this.criteriaEvent = null;
        // reset users list
        this.axUsersAndGroupsList = [];
        this.axSlackChannelsList = [];
        this.outsideUsers = [];
        this.filteredOutsideUsers = [];
        // reset rule
        this.rule = null;
        // move to first step
        this.isFirstStep = true;
        this.isSlackIntegrationConfigured = false;
    }

    public onEventTypeChange(event: string[]) {
        this.notificationCreationForm.controls['channels'].setValue(event);
    }

    public onEventSeverityChange(event: string[]) {
        this.notificationCreationForm.controls['severities'].setValue(event);
    }

    public updateUsersList(users: string[]) {
        this.axUsersAndGroupsList = users;
        this.updateRecipientsList();
        this.clearValidators();
    }

    public updateSlackChannelsList(channels: string[]) {
        this.axSlackChannelsList = channels.map(channel => `${channel}@slack`);
        this.updateRecipientsList();
        this.clearValidators();
    }

    public updateRecipientsList() {
        let val = this.axUsersAndGroupsList.concat(this.axSlackChannelsList, this.filteredOutsideUsers);
        let uniqueList = val.filter((value, index, self) => self.indexOf(value) === index );
        this.notificationCreationForm.controls['recipients'].setValue(uniqueList || []);
    }

    public openUserSelectorPanel() {
        this.isVisibleUserSelectorPanel = true;
    }

    public openSlackChannelPanel() {
        this.isVisibleSlackPanel = true;
    }

    public closeUserSelectorPanel() {
        this.isVisibleUserSelectorPanel = false;
    }

    public closeSlackChannelPalen() {
        this.isVisibleSlackPanel = false;
    }

    public updateOutsideUsers(users: string) {
        // i moved scope up, to be able to validate on click submit btn
        this.outsideUsers = users.split(',');
        this.filteredOutsideUsers = this.outsideUsers.filter(user => CustomRegex.emailPattern.test(user.trim())).map(user => `${user}@user`);
        this.updateRecipientsList();
        this.clearValidators();
    }

    private async getChannels() {
        return await this.notificationService.getChannels();
    }

    private async getSeverities() {
        return await this.notificationService.getSeverities();
    }

    private async createRule() {
        let rule: Rule = this.notificationCreationForm.value;
        try {
            await this.notificationService.createRule(rule);
            this.notificationsService.success(`The rule: ${rule.name} was successfully created.`);
        } catch (err) {
            this.notificationsService.error(`The rule: ${rule.name} wasn't created correct. Something went wrong.`);
        }
    }

    private async updateRule() {
        let rule: Rule = this.notificationCreationForm.value;
        rule.rule_id = this.rule.rule_id;
        try {
            await this.notificationService.updateRule(rule);
            this.notificationsService.success(`The rule: ${rule.name} was successfully updated.`);
        } catch (err) {
            this.notificationsService.error(`The rule: ${rule.name} wasn't updated correct. Something went wrong.`);
        }
    }

    private async getAndAddCriteria() {
        if (this.isNotificationCriteriaPanelVisible && !this.criteriaEvent) {
            this.isVisibleCriteriaLoader = true;
            try {
                this.channels = await this.getChannels();
                this.severities = await this.getSeverities();

                this.eventTypes.items = this.channels.map(item => {
                    return {name: ViewUtils.capitalizeFirstLetter(item), value: item, checked: false};
                });

                this.eventSeverity.items = this.severities.map(item => {
                    return {name: ViewUtils.capitalizeFirstLetter(item), value: item, checked: false};
                });

                this.addCriteria();
                this.isVisibleCriteriaLoader = false;
            } catch (err) {
                this.isVisibleCriteriaLoader = false;
            }
        }
    }

    private clearValidators() {
        this.validationMessages.ruleName.show = false;
        this.validationMessages.eventType.show = false;
        this.validationMessages.eventSeverity.show = false;
        this.validationMessages.missingRecipients.show = false;
        this.validationMessages.wrongFormatRecipients.show = false;
    }

    private checkIfSlackIntergrationConfigured() {
        this.toolService.getToolsAsync({type: 'slack'}, true).toPromise().then(res => {
            this.isSlackIntegrationConfigured = !!res.data.length;
        });
    }

    private shareUsersToLists() {
        if (!this.rule) {
            return;
        }

        this.axSlackChannelsList = this.rule.recipients.filter(recipient => recipient.indexOf('@slack') !== -1).sort();
        this.axUsersAndGroupsList = this.rule.recipients.filter(recipient => recipient.indexOf('@slack') === -1 && recipient.indexOf('@user') === -1).sort();
        this.filteredOutsideUsers = this.rule.recipients.filter(recipient => recipient.indexOf('@user') !== -1).sort();
        this.outsideUsers = this.filteredOutsideUsers.map(user => user.substring(0, user.indexOf('@user')));
    }
}
