import { Component, OnInit, ViewChild } from '@angular/core';

import * as models from '../../models';
import { WorkflowsService, EventsService } from '../../services';
import { NODE_PHASE } from '../../models';
import { DropDownComponent } from 'ui-lib/src/components';

@Component({
  selector: 'ax-workflows-list-page',
  templateUrl: './workflows-list-page.component.html',
  styleUrls: ['./workflows-list-page.component.scss']
})
export class WorkflowsListPageComponent implements OnInit {

  public workflowList: models.WorkflowList;
  public toolbarFilters: any = {
    data: [],
    model: [],
  };
  private pageTitle = 'Timeline';

  @ViewChild(DropDownComponent)
  private dropdown: DropDownComponent;

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

  public async toggleFilter(option) {
    if (this.toolbarFilters.model.indexOf(option.value) > -1) {
      this.toolbarFilters.model.splice(this.toolbarFilters.model.indexOf(option.value), 1);
    } else {
      this.toolbarFilters.model.push(option.value);
    }

    this.dropdown.close();
    this.workflowList = await this.workflowsService.getWorkflows(this.toolbarFilters.model);
  }

  public get hasFilters(): boolean {
    return this.toolbarFilters.data.filter(item => {
      return this.toolbarFilters.model.indexOf(item.value) > -1;
    }).length > 0;
  }
}
