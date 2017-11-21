import { Component, Input, OnChanges } from '@angular/core';

import { Workflow } from '../../models';

@Component({
    selector: 'app-workflow-details-box',
    templateUrl: './workflow-details-box.html',
})
export class WorkflowDetailsBoxComponent {
    @Input()
    workflow: Workflow;
}
