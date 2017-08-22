import {Directive, Input, Renderer, ElementRef} from '@angular/core';

import {TaskStatus} from '../../model';

@Directive({
    selector: '[ax-status-icon]'
})
export class StatusIconDirective {

    private statusVal: number;

    constructor(private renderer: Renderer, private el: ElementRef) {
    }

    @Input()
    public set status(val: number) {
        this.getStatusClasses(this.statusVal).forEach((item: string) => this.renderer.setElementClass(this.el.nativeElement, item, false));
        this.getStatusClasses(val).forEach((item: string) => this.renderer.setElementClass(this.el.nativeElement, item, true));
        this.statusVal = val;
    };

    private getStatusClasses(status: number): string[] {
        let styleClasses = [];

        switch (status) {
            case TaskStatus.Cancelled:
                styleClasses = ['fa-exclamation-circle', 'status-icon--cancelled'];
                break;
            case TaskStatus.Failed:
                styleClasses = ['fa-times-circle', 'status-icon--failed'];
                break;
            case TaskStatus.Success:
                styleClasses = ['fa-check-circle', 'status-icon--success'];
                break;
            case TaskStatus.Waiting:
                styleClasses = ['fa-exclamation-circle', 'status-icon--waiting'];
                break;
            case TaskStatus.Running:
            case TaskStatus.Canceling:
                styleClasses = ['fa-circle-o-notch', 'status-icon--running', 'status-icon--spin'];
                break;
            case TaskStatus.Init:
                styleClasses = ['fa-clock-o', 'status-icon--init'];
                break;
        }

        return styleClasses;
    }
}
