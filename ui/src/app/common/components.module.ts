import { NgModule } from '@angular/core';

import { DurationPipe } from './duration.pipe';
import { TimestampPipe } from './timestamp.pipe';
import { WorkflowStatusPipe } from './workflow-status.pipe';
import { StatusIconDirective } from './status-icon/status-icon.directive';

const components = [
  DurationPipe,
  TimestampPipe,
  WorkflowStatusPipe,
  StatusIconDirective,
];

@NgModule({
  declarations: components,
  exports: components
})
export class ComponentsModule {}
