import { NgModule } from '@angular/core';
import { CommonModule } from '@angular/common';

import { BaseModule } from '../common';

import { WorkflowsRoutingModule } from './workflows-routing.module';
import { WorkflowsListPageComponent } from './workflows-list-page/workflows-list-page.component';
import { WorkflowDetailsPageComponent } from './workflow-details-page/workflow-details-page.component';
import { WorkflowDetailsBoxComponent } from './workflow-details-box/workflow-details-box.component';
import { WorkflowItemComponent } from './workflows-list-page/workflow-item/workflow-item.component';
import { WorkflowStepDetailsBoxComponent } from './workflow-step-details-box/workflow-step-details-box.component';
import { WorkflowStepsComponent } from './workflows-list-page/workflow-steps/workflow-steps.component';

@NgModule({
  imports: [
    CommonModule,
    WorkflowsRoutingModule,
    BaseModule,
  ],
  declarations: [
    WorkflowsListPageComponent,
    WorkflowDetailsPageComponent,
    WorkflowDetailsBoxComponent,
    WorkflowItemComponent,
    WorkflowStepsComponent,
    WorkflowStepDetailsBoxComponent,
  ]
})
export class WorkflowsModule { }
