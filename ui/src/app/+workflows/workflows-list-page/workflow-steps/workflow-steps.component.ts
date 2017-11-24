import { Component, OnInit, Input } from '@angular/core';

import * as models from '../../../models';

@Component({
  selector: 'ax-workflow-steps',
  templateUrl: './workflow-steps.component.html',
  styleUrls: ['./workflow-steps.component.scss']
})
export class WorkflowStepsComponent implements OnInit {

  @Input()
  public workflow: models.WorkflowList;

  public ngOnInit() {
  }
}
