import { Component, Output, EventEmitter } from '@angular/core';

@Component({
    selector: 'ax-setup-job-notifications',
    templateUrl: './setup-job-notifications.html',
    styles: [ require('./setup-job-notifications.scss') ],
})
export class SetupJobNotificationsComponent {
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
            name: '',
        },
        isVisible: false,
        isStaticList: true,
        isDisplayedInline: true
    };

    public onEventTypeChange(event: string[]) {
        console.log('onEventTypeChange', event);
    }
}
