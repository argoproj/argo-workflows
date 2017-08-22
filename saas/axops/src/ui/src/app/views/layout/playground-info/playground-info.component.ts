import { Component, Input } from '@angular/core';
import { Router } from '@angular/router';

import { PlaygroundTaskInfo } from '../../../services';

@Component({
    selector: 'ax-playground-info',
    templateUrl: './playground-info.html',
    styles: [ require('./playground-info.scss')],
})
export class PlaygroundInfoComponent {

    public showVideoPopup = false;

    @Input()
    public taskInfo: PlaygroundTaskInfo;

    constructor(private router: Router) {
    }

    public get showBackUrl(): boolean {
        return this.taskInfo && !this.router.url.startsWith(this.taskInfo.backUrl);
    }
}
