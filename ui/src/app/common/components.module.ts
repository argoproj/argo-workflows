import { NgModule } from '@angular/core';
import { CommonModule } from '@angular/common';

import { DurationPipe } from './duration.pipe';
import { TimestampPipe } from './timestamp.pipe';
import { ShortTimePipe } from './short-time.pipe';
import { WorkflowStatusPipe } from './workflow-status.pipe';
import { WorkflowTreeComponent } from './workflow-tree/workflow-tree.component';
import { WorkflowSubtreeComponent } from './workflow-tree/workflow-subtree.component';
import { WorkflowTreeNodeComponent } from './workflow-tree/workflow-tree-node.component';

const components = [
  DurationPipe,
  TimestampPipe,
  WorkflowStatusPipe,
  ShortTimePipe,
  WorkflowTreeComponent,
  WorkflowSubtreeComponent,
  WorkflowTreeNodeComponent,
];

@NgModule({
  declarations: components,
  exports: components,
  imports: [
    CommonModule,
  ]
})
export class ComponentsModule {}
