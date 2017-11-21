import { NgModule } from '@angular/core';

import { DurationPipe } from './duration.pipe';
import { TimestampPipe } from './timestamp.pipe';
import { WorkflowStatusPipe } from './workflow-status.pipe';

const components = [
  DurationPipe,
  TimestampPipe,
  WorkflowStatusPipe,
];

@NgModule({
  declarations: components,
  exports: components
})
export class ComponentsModule {}
