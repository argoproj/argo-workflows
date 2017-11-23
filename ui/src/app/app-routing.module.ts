import { NgModule } from '@angular/core';
import { Routes, RouterModule } from '@angular/router';

const routes: Routes = [
  { path: 'timeline', loadChildren: 'app/+workflows/workflows.module#WorkflowsModule' },
  { path: 'test', loadChildren: 'app/+workflows/workflows.module#WorkflowsModule' },
  { path: 'help', loadChildren: 'app/+workflows/workflows.module#WorkflowsModule' },
  ];

@NgModule({
  imports: [RouterModule.forRoot(routes)],
  exports: [RouterModule]
})
export class AppRoutingModule { }
