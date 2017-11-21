import { Component, OnInit } from '@angular/core';
import { ActivatedRoute } from '@angular/router';

import { WorkflowsService } from '../../services';
import * as models from '../../models';

@Component({
  selector: 'app-workflow-details-page',
  templateUrl: './workflow-details-page.html',
  styleUrls: ['./workflow-details.scss']
})
export class WorkflowDetailsPageComponent implements OnInit {

  public workflow: models.Workflow;

  constructor(private workflowsService: WorkflowsService, private route: ActivatedRoute) {}

  public ngOnInit() {
    this.route.params.subscribe(async params => {
      this.workflow = await this.workflowsService.getWorkflow(params['name']);
    });
  }
}
