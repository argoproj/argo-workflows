import { Component, OnInit } from '@angular/core';

import * as models from '../../models';
import { WorkflowsService, EventsService } from '../../services';

@Component({
  selector: 'ax-workflows-list-page',
  templateUrl: './workflows-list-page.component.html',
  styleUrls: ['./workflows-list-page.component.scss']
})
export class WorkflowsListPageComponent implements OnInit {

  private pageTitle = 'Timeline';

  public workflowList: models.WorkflowList;

  constructor(private workflowsService: WorkflowsService, private eventsService: EventsService) { }

  public async ngOnInit() {
    this.eventsService.setPageTitle.emit(this.pageTitle);
    this.workflowList = await this.workflowsService.getWorkflows();
  }
}
