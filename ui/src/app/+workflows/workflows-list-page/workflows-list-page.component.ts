import { Component, OnInit } from '@angular/core';

import * as models from '../../models';
import { WorkflowsService, EventsService } from '../../services';
import { NODE_PHASE } from '../../models';
import { DropDownComponent } from 'ui-lib/src/components';
import { ToolbarFilters } from '../../common/toolbar';

@Component({
  selector: 'ax-workflows-list-page',
  templateUrl: './workflows-list-page.component.html',
  styleUrls: ['./workflows-list-page.component.scss']
})
export class WorkflowsListPageComponent implements OnInit {

  public workflowList: models.WorkflowList;
  public toolbarFilters: ToolbarFilters = {
    data: [],
    model: [],
  };
  private pageTitle = 'Timeline';

  constructor(private workflowsService: WorkflowsService, private eventsService: EventsService) { }

  public async ngOnInit() {
    for (status in NODE_PHASE) {
      this.toolbarFilters.data.push({
        name: NODE_PHASE[status],
        value: NODE_PHASE[status],
      });
    }

    this.eventsService.setPageTitle.emit(this.pageTitle);
    this.workflowList = await this.workflowsService.getWorkflows(this.toolbarFilters.model);
  }

  public async toggleFilter(model: string[]) {
    this.toolbarFilters.model = model;

    this.workflowList = await this.workflowsService.getWorkflows(this.toolbarFilters.model);
  }
}
