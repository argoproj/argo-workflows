import { Component, OnInit, OnDestroy } from '@angular/core';
import { Subscription } from 'rxjs/Subscription';

import * as models from '../../models';
import { WorkflowsService, EventsService } from '../../services';
import { DropDownComponent } from 'ui-lib/src/components';
import { ToolbarFilters } from '../../common/toolbar';

@Component({
  selector: 'ax-workflows-list-page',
  templateUrl: './workflows-list-page.component.html',
  styleUrls: ['./workflows-list-page.component.scss']
})
export class WorkflowsListPageComponent implements OnInit, OnDestroy {

  public workflowList: models.WorkflowList;
  public toolbarFilters: ToolbarFilters = {
    data: [],
    model: [],
  };

  private pageTitle = 'Timeline';
  private workflowByName = new Map<String, models.Workflow>();
  private subscription: Subscription;

  constructor(private workflowsService: WorkflowsService, private eventsService: EventsService) { }

  public ngOnDestroy() {
    this.ensureUnsubscribed();
  }

  public async ngOnInit() {
    for (const status of Object.keys(models.NODE_PHASE)) {
      this.toolbarFilters.data.push({
        name: models.NODE_PHASE[status],
        value: models.NODE_PHASE[status],
      });
    }

    this.eventsService.setPageTitle.emit(this.pageTitle);
    this.refreshWorkflows();
  }

  public async toggleFilter(model: string[]) {
    this.toolbarFilters.model = model;

    this.refreshWorkflows();
  }

  private async refreshWorkflows() {
    this.workflowList = await this.workflowsService.getWorkflows(this.toolbarFilters.model);
    this.workflowByName.clear();
    this.workflowList.items.forEach(item => this.workflowByName.set(item.metadata.name, item));
    this.ensureUnsubscribed();
    this.subscription = this.workflowsService.getWorkflowsStream().subscribe(workflow => {
      const existingWorkflow = this.workflowByName.get(workflow.metadata.name);
      if (existingWorkflow) {
        Object.assign(existingWorkflow, workflow);
      } else {
        this.workflowByName.set(workflow.metadata.name, workflow);
        this.workflowList.items.unshift(workflow);
      }
    });
  }

  private ensureUnsubscribed() {
    if (this.subscription) {
      this.subscription.unsubscribe();
      this.subscription = null;
    }
  }
}
