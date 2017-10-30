import { Component, EventEmitter, Input, Output } from '@angular/core';

import { TrackingService, AuthenticationService, ViewPreferencesService } from '../../../../services';

@Component({
    selector: 'ax-feedback-panel',
    templateUrl: './feedback-panel.html',
    styles: [ require('./feedback-panel.scss') ],
})
export class FeedbackPanelComponent {

    constructor(private trackingService: TrackingService, private authenticationService: AuthenticationService, private viewPreferencesService: ViewPreferencesService) {}

    @Input()
    public show: boolean;

    @Output()
    public onClose: EventEmitter<null> = new EventEmitter();

    public selectedOption: string;
    public formOption: string;
    public comments: string;

    public select(option: string) {
        this.selectedOption = option;
        if (option === 'good') {
            this.submitFeedback();
        }
    }

    public selectFormOption(option: string) {
        this.formOption = option;
    }

    public async submitFeedback() {
        await this.trackingService.sendUsageEvent('first-user-job', this.authenticationService.getUsername(), {
            rating: this.selectedOption,
            area: this.formOption,
            comments: this.comments,
        });
        await this.viewPreferencesService.updateViewPreferences(preferences => preferences.firstJobFeedbackStatus = 'feedback-submitted');
        this.close();
    }

    public close() {
        this.show = false;
        this.onClose.emit();
    }
}
