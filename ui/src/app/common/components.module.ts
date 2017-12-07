import { NgModule } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule, ReactiveFormsModule } from '@angular/forms';

import { GuiComponentsModule } from 'ui-lib/src/components';

import { DurationPipe } from './duration.pipe';
import { TimestampPipe } from './timestamp.pipe';
import { ShortTimePipe } from './short-time.pipe';
import { TruncateToPipe } from './trancate-to.pipe';
import { WorkflowStatusPipe } from './workflow-status.pipe';
import { StatusIconDirective } from './status-icon/status-icon.directive';
import { WorkflowTreeComponent } from './workflow-tree/workflow-tree.component';
import { WorkflowSubtreeComponent } from './workflow-tree/workflow-subtree.component';
import { WorkflowTreeNodeComponent } from './workflow-tree/workflow-tree-node.component';
import { ArtifactsComponent } from './artifacts/artifacts.component';
import { YamlViewerComponent } from './yaml-viewer/yaml-viewer.component';
import { SysConsoleComponent } from './sys-console/sys-console.component';

const components = [
  ArtifactsComponent,
  DurationPipe,
  TimestampPipe,
  WorkflowStatusPipe,
  StatusIconDirective,
  ShortTimePipe,
  TruncateToPipe,
  WorkflowTreeComponent,
  WorkflowSubtreeComponent,
  WorkflowTreeNodeComponent,
  YamlViewerComponent,
  SysConsoleComponent
];

@NgModule({
  declarations: components,
  exports: components,
  imports: [
    CommonModule,
    GuiComponentsModule,
    FormsModule,
    ReactiveFormsModule
  ]
})
export class ComponentsModule {}
