import { NgModule } from '@angular/core';
import { CommonModule } from '@angular/common';

import { BaseModule } from '../common';

import { WorkflowsRoutingModule } from './workflows-routing.module';
import { WorkflowsListPageComponent } from './workflows-list-page/workflows-list-page.component';
import { WorkflowDetailsPageComponent } from './workflow-details-page/workflow-details-page.component';

@NgModule({
  imports: [
    CommonModule,
    WorkflowsRoutingModule,
    BaseModule,
  ],
  declarations: [
    WorkflowsListPageComponent,
    WorkflowDetailsPageComponent
  ]
})
export class WorkflowsModule { }
