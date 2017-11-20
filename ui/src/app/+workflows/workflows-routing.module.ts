import { NgModule } from '@angular/core';
import { Routes, RouterModule } from '@angular/router';
import { WorkflowsListPageComponent } from './workflows-list-page/workflows-list-page.component';

const routes: Routes = [{
    path: '', component: WorkflowsListPageComponent,
}];

@NgModule({
  imports: [RouterModule.forChild(routes)],
  exports: [RouterModule]
})
export class WorkflowsRoutingModule { }
