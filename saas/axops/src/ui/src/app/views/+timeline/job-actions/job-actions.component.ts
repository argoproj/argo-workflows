import { Component, OnChanges, Input } from '@angular/core';

import { Task, TaskStatus } from '../../../model';
import { JobsService } from '../jobs.service';
import { DropdownMenuSettings } from 'argo-ui-lib/src/components';

@Component({
    selector: 'ax-job-actions',
    template: `<ax-dropdown-menu [settings]="menuSettings">
                   <ng-content *ngIf="menuSettings && menuSettings.menu.length > 0"></ng-content>
               </ax-dropdown-menu>`,
})
export class JobActionsComponent implements OnChanges {
    @Input()
    task: Task;
    @Input()
    stepName: string;
    @Input()
    allowResubmit: boolean = true;
    @Input()
    rootTask: Task;

    menuSettings: DropdownMenuSettings;

    constructor(private jobsService: JobsService) {
    }

    ngOnChanges() {
        if (this.task && this.rootTask) {
            let step = this.jobsService.getSelectedStep(this.task, this.rootTask);
            let menuSettings = this.jobsService.getActionMenuSettings(this.task, this.rootTask);
            // add artifacts download links to dropdown if task is finished
            if (this.task.status ===  TaskStatus.Skipped ||
                this.task.status ===  TaskStatus.Cancelled ||
                this.task.status ===  TaskStatus.Failed ||
                this.task.status ===  TaskStatus.Success ) {
                menuSettings.menu = menuSettings.menu.concat(this.jobsService.getArtifactMenuItems(this.task, step));
            }
            this.menuSettings = menuSettings;
        }
    }
}
