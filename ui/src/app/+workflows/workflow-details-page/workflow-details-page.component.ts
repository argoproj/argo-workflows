import { Component, OnInit } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';

import { WorkflowsService } from '../../services';
import * as models from '../../models';

@Component({
  selector: 'ax-workflow-details-page',
  templateUrl: './workflow-details-page.html',
  styleUrls: ['./workflow-details.scss']
})
export class WorkflowDetailsPageComponent implements OnInit {

  public workflow: models.Workflow;
  public selectedTab = 'summary';

  constructor(private workflowsService: WorkflowsService, private route: ActivatedRoute, private router: Router) {}

  public tabChange(tab: string) {
    this.router.navigate(['.', { tab }], { relativeTo: this.route });
  }

  public ngOnInit() {
    this.route.params.map(params => params['name']).distinct().subscribe(async name => {
      this.workflow = await this.workflowsService.getWorkflow(name);
    });
    this.route.params.map(params => params['tab']).distinct().subscribe(tab => {
      this.selectedTab = tab || 'summary';
    });
  }

  public getProgressClasses(stepStatus: string) {
    const status = stepStatus === models.NODE_STATUS.FAILED ? 'failed' : 'running';
    let percentage = 0;
    switch (stepStatus) {
      case models.NODE_STATUS.INIT:
        percentage = 0;
        break;
      case models.NODE_STATUS.RUNNING:
        percentage = 50;
        break;
      case models.NODE_STATUS.SUCCEEDED:
      case models.NODE_STATUS.SKIPPED:
      case models.NODE_STATUS.FAILED:
      case models.NODE_STATUS.ERROR:
        percentage = 100;
        break;
    }
    return [
        'workflow-details__node-progress', `workflow-details__node-progress--${percentage.toFixed()}-${status}`
    ].join(' ');
  }
}
