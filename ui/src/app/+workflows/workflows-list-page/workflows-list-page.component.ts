import { Component, OnInit } from '@angular/core';

import * as models from '../../models';
import { WorkflowsService } from '../../services';

@Component({
  selector: 'app-workflows-list-page',
  templateUrl: './workflows-list-page.component.html',
  styleUrls: ['./workflows-list-page.component.scss']
})
export class WorkflowsListPageComponent implements OnInit {

  public workflowList: models.WorkflowList;

  constructor(private workflowsService: WorkflowsService) { }

  public async ngOnInit() {
    this.workflowList = await this.workflowsService.getWorkflows();
  }
}
