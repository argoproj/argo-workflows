import { Component, Input } from '@angular/core';
import { DomSanitizer } from '@angular/platform-browser';
import { NotificationEvent } from '../../../../model';
import { CustomRegex } from '../../../../common';
import { ModalService, NotificationService, AuthenticationService } from '../../../../services';

@Component({
    selector: 'ax-notification-details',
    templateUrl: './notification-details.html',
    styles: [ require('./notification-details.scss') ],
})
export class NotificationDetailsComponent {

    @Input()
    public notification: NotificationEvent;

    constructor(
        private modalService: ModalService,
        private notificationService: NotificationService,
        private authenticationService: AuthenticationService,
        private domSanitizer: DomSanitizer ) {
    }

    public get attributes(): string[] {
        return this.notification && this.notification.detail && Object.keys(this.notification.detail) || [];
    }

    public acknowledge() {
        this.modalService.showModal('Acknowledge notification?', 'Are you sure you to acknowledge notification?').subscribe(async confirmed => {
            if (confirmed) {
                 let notification = await this.notificationService.acknowledgeNotification(this.notification.event_id);
                 Object.assign(this.notification, notification);
            }
        });
    }

    public isAcknowledgeByCurrentUser(): boolean {
        return this.notification.acknowledged_by === this.authenticationService.getUsername();
    }

    public notificationAttributeValue(attributeName: string) {
        let val: string = this.notification.detail[attributeName] ? this.notification.detail[attributeName].toString() : '';
        let formattedVal: string;
        if (val.match(CustomRegex.url)) {
            formattedVal = `<a href="${val}">${val}</a>`;
        } else if (val.match(CustomRegex.email)) {
            formattedVal = `<a href="mailto:${val}">${val}</a>`;
        } else {
            formattedVal = val;
        }
        return this.domSanitizer.bypassSecurityTrustHtml(formattedVal);
    }
}
