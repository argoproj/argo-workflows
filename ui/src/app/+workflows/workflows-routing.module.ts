import { NgModule } from '@angular/core';
import { Routes, RouterModule } from '@angular/router';
import { WorkflowsListPageComponent } from './workflows-list-page/workflows-list-page.component';
import { WorkflowDetailsPageComponent } from './workflow-details-page/workflow-details-page.component';

const routes: Routes = [{
  path: '', component: WorkflowsListPageComponent, pathMatch: 'full'
}, {
  path: ':name', component: WorkflowDetailsPageComponent,
}];

@NgModule({
  imports: [RouterModule.forChild(routes)],
  exports: [RouterModule]
})
export class WorkflowsRoutingModule { }
