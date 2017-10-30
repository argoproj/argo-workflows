import { Component, EventEmitter, Input, Output } from '@angular/core';

@Component({
    selector: 'ax-feedback-panel',
    templateUrl: './feedback-panel.html',
    styles: [ require('./feedback-panel.scss') ],
})
export class FeedbackPanelComponent {

    @Input()
    public show: boolean;

    @Output()
    public onClose: EventEmitter<null> = new EventEmitter();

    public selectedOption: string;
    public formOption: string;

    public select(option: string) {
        this.selectedOption = option;
    }

    public selectFormOption(option: string) {
        this.formOption = option;
    }

    public close() {
        this.show = false;
        this.onClose.emit();
    }
}
