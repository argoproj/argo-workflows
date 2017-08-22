import {Component} from '@angular/core';
import {NotificationService} from '../../services';

@Component({
    selector: 'ax-notification',
    templateUrl: './notification.html',
    styles: [ require('./notification.scss') ],
})

export class NotificationComponent {
    private notificationVissible: boolean = false;
    private theHtmlString: string = '';

    constructor(private _notificationEventService: NotificationService) {
        // TODO add removing notification if change url
        _notificationEventService.showNotification.subscribe((response) => {
            this.notificationVissible = true;
            this.theHtmlString = response.message;

            setTimeout(() => {
                this.notificationVissible = false;
            }, 5000);
        });
    }

    closeNotification() {
        this.notificationVissible = false;
    }
}
