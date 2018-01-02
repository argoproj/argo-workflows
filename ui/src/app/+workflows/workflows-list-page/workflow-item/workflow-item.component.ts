import { Component, Input } from '@angular/core';
import { Workflow } from '../../../models/index';

@Component({
  selector: 'ax-workflow-item',
  templateUrl: './workflow-item.component.html',
  styleUrls: ['./workflow-item.component.scss']
})
export class WorkflowItemComponent {

  @Input()
  public workflow: Workflow;

  public get status(): string {
    return this.workflow && this.workflow.status && this.workflow.status.phase;
  }
}
