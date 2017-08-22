import { Component, Input } from '@angular/core';
import { NotificationEvent } from '../../../../model';

@Component({
    selector: 'ax-notification-summary',
    templateUrl: './notification-summary.html',
    styles: [ require('./notification-summary.scss') ],
})
export class NotificationSummaryComponent {

    @Input()
    public notification: NotificationEvent;
}
