import { Pipe, PipeTransform } from '@angular/core';

import * as models from '../models';

@Pipe({
  name: 'workflowStatus'
})
export class WorkflowStatusPipe implements PipeTransform {
  public transform(value: models.Workflow, ...args: any[]) {
    return value.status.phase;
  }
}
