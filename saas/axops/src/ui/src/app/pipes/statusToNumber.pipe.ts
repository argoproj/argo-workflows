import { Pipe, PipeTransform } from '@angular/core';

import { TaskStatus } from '../model';

export const TASK_STATUSES = {
    CANCELLED: 'Cancelled',
    FAILED: 'Failed',
    SUCCESSFUL: 'Successful',
    QUEUED: 'Queued',
    IN_PROGRESS: 'In-Progress',
};

@Pipe({
    name: 'statusToNumber'
})

export class StatusToNumberPipe implements PipeTransform {
    transform(value: string, args?: any[]) {
        let status: number;

        switch (value) {
            case TASK_STATUSES.CANCELLED:
                status = TaskStatus.Cancelled;
                break;
            case TASK_STATUSES.FAILED:
                status = TaskStatus.Failed;
                break;
            case TASK_STATUSES.SUCCESSFUL:
                status = TaskStatus.Success;
                break;
            case TASK_STATUSES.QUEUED:
                status = TaskStatus.Waiting;
                break;
            case TASK_STATUSES.IN_PROGRESS:
                status = TaskStatus.Running;
                break;
        }

        return status;
    }
}
