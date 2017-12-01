import * as moment from 'moment';
import { Component, Input, OnChanges, SimpleChanges } from '@angular/core';

import * as models from '../../models';
import { WorkflowTree } from '../../common';

@Component({
  selector: 'ax-workflow-step-summary-box',
  templateUrl: './workflow-step-summary-box.html',
})
export class WorkflowStepSummaryBoxComponent implements OnChanges {

  public step: models.WorkflowStep;
  public status: models.NodeStatus;

  @Input()
  public nodeName: string;

  @Input()
  public workflowTree: WorkflowTree;

  public ngOnChanges(changes: SimpleChanges): void {
    if (this.nodeName && this.workflowTree) {
      this.status = this.workflowTree.workflow.status.nodes[this.nodeName];
      const stepName = this.status.name;
      this.step = this.workflowTree.getStepByName(stepName);
    }
  }

  public get stepDuration() {
    if (this.status) {
      const endTime = this.status.finishedAt ? moment(this.status.finishedAt) : moment();
      return endTime.diff(moment(this.status.startedAt)) / 1000;
    }
    return null;
  }
}
