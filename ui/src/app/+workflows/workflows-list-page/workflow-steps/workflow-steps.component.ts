import { Component, Input } from '@angular/core';

import * as models from '../../../models';
import { Workflow, NODE_PHASE } from '../../../models';

@Component({
  selector: 'ax-workflow-steps',
  templateUrl: './workflow-steps.component.html',
  styleUrls: ['./workflow-steps.component.scss']
})
export class WorkflowStepsComponent {

  private _workflow: Workflow;

  public steps: { isSucceeded: boolean; isFailed: boolean; isRunning: boolean; name: string } [] = [];

  @Input()
  public set workflow(val: models.Workflow) {
    this._workflow = val;
    if (val) {
      const entryPointTemplate = val.spec.templates.find(template => template.name === val.spec.entrypoint);
      const phase = this._workflow.status.nodes[val.metadata.name].phase;
      let isSucceeded = false;
      let isFailed = false;
      let isRunning = false;
      if (phase === NODE_PHASE.RUNNING) {
        isRunning = true;
      } else {
        isSucceeded = phase === NODE_PHASE.SUCCEEDED;
        isFailed = !isSucceeded;
      }
      this.steps = (entryPointTemplate.steps || []).map(
        group => group[0]).map(step => ({ name: step.name, isSucceeded, isFailed, isRunning }));
    } else {
      this.steps = [];
    }
  }

  public get workflow(): models.Workflow {
    return this._workflow;
  }
}
