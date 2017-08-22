import { Pipe, PipeTransform } from '@angular/core';

import { TaskStatus } from '../model';

@Pipe({
    name: 'status'
})

export class StatusPipe implements PipeTransform {
    transform(value: number, args?: any[]) {
        let status = '';

        switch (value) {
            case TaskStatus.Cancelled:
                status = 'Cancelled';
                break;
            case TaskStatus.Failed:
                status = 'Failed';
                break;
            case TaskStatus.Success:
                status = 'Successful';
                break;
            case TaskStatus.Waiting:
                status = 'Waiting';
                break;
            case TaskStatus.Running:
                status = 'In Progress';
                break;
            case TaskStatus.Init:
                status = 'Queued';
                break;
            case TaskStatus.Canceling:
                status = 'Canceling';
                break;
        }

        return status;
    }
}
