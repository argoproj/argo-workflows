import { Component, Input, Output, EventEmitter } from '@angular/core';
import { WorkflowTree } from '../../common';

import { WorkflowsService } from '../../services';

@Component({
  selector: 'ax-workflow-step-details-box',
  templateUrl: './workflow-step-details-box.html',
  styleUrls: ['./workflow-step-details-box.scss']
})
export class WorkflowStepDetailsBoxComponent {

  constructor(private workflowsService: WorkflowsService) {
  }

  @Input()
  public tab: string;

  @Input()
  public nodeName: string;

  @Input()
  public workflowTree: WorkflowTree;

  @Output()
  public onTabSelected = new EventEmitter<string>();

  public getLogsSource() {
    return {
      loadLogs: () => {
        return this.nodeName && this.workflowsService.getStepLogs(this.workflowTree.workflow.metadata.namespace, this.nodeName);
      },
      getKey() {
        return this.nodeName;
      }
    };
  }
}
