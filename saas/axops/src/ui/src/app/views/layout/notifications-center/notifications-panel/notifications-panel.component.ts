import { Component, Input, Output, EventEmitter } from '@angular/core';

import { NotificationService, AuthenticationService } from '../../../../services';
import { NotificationEvent } from '../../../../model';

@Component({
    selector: 'ax-notifications-panel',
    templateUrl: './notifications-panel.html',
    styles: [require('./notifications-panel.scss')],
})
export class NotificationsPanelComponent {

    @Input()
    public get show(): boolean {
        return this.showPanel;
    }

    public set show(val: boolean) {
        if (this.showPanel !== val && val) {
            this.reloadEvents();
        }
        this.showPanel = val;
    }

    @Output()
    public onClose: EventEmitter<null> = new EventEmitter<null>();

    public categories: { value: string, title: string }[] = [
        { value: 'critical', title: 'Critical' },
        { value: 'warning', title: 'Warning' },
        { value: 'all', title: 'All' }
    ];
    public notifications: NotificationEvent[] = [];
    public selectedCategory: string = 'all';
    public selectedNotification: NotificationEvent;
    public loading: boolean = false;

    private showPanel: boolean;
    private canScroll: boolean = false;
    private offset: number = 0;
    private readonly bufferSize: number = 20;

    constructor(private notificationsService: NotificationService, private authenticationService: AuthenticationService) {
    }

    public categoryChanged(category: string) {
        this.selectedCategory = category;
        this.reloadEvents();
    }

    public showDetails(notification: NotificationEvent) {
        this.selectedNotification = notification;
    }

    public close() {
        this.showDetails(null);
        this.onClose.next();
    }

    public onScroll() {
        if (this.canScroll) {
            this.canScroll = false;
            this.loadEvents();
        }
    }

    public reloadEvents() {
        this.offset = 0;
        this.notifications = [];
        this.loadEvents();
    }

    private async loadEvents() {
        this.loading = true;
        try {
            let user = await this.authenticationService.getCurrentUser();
            let notifications = await this.notificationsService.getEvents({
                recipient: user.username,
                limit: this.bufferSize,
                offset: this.offset,
                severity: this.selectedCategory === 'all' ? null : this.selectedCategory,
            });
            this.notifications = this.notifications.concat(notifications);
            this.offset = this.offset + notifications.length;
            this.canScroll = notifications.length === this.bufferSize;
        } finally {
            this.loading = false;
        }
    }
}
