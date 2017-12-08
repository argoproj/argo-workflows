import { Directive, Input, Renderer, ElementRef } from '@angular/core';

import { NODE_PHASE } from '../../models';

@Directive({
    selector: '[axStatusIcon]'
})
export class StatusIconDirective {

    private statusVal: string;

    constructor(private renderer: Renderer, private el: ElementRef) {
    }

    @Input()
    public set status(val: string) {
        this.getStatusClasses(this.statusVal).forEach((item: string) => this.renderer.setElementClass(this.el.nativeElement, item, false));
        this.getStatusClasses(val).forEach((item: string) => this.renderer.setElementClass(this.el.nativeElement, item, true));
        this.statusVal = val;
    }

    private getStatusClasses(status: string): string[] {
        let styleClasses = [];

        switch (status) {
            case NODE_PHASE.ERROR:
            case NODE_PHASE.FAILED:
                styleClasses = ['fa-times-circle', 'status-icon--failed'];
                break;
            case NODE_PHASE.SUCCEEDED:
                styleClasses = ['fa-check-circle', 'status-icon--success'];
                break;
            case NODE_PHASE.RUNNING:
                styleClasses = ['fa-circle-o-notch', 'status-icon--running', 'status-icon--spin'];
                break;
            default:
                styleClasses = ['fa-clock-o', 'status-icon--init'];
                break;
        }

        return styleClasses;
    }
}
