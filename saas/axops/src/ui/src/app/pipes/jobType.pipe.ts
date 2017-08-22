import { Pipe, PipeTransform } from '@angular/core';
import { TaskStatus } from '../model';

@Pipe({
    name: 'jobType'
})

export class JobTypePipe implements PipeTransform {
    transform(value: number, args?: any[]) {
        let type = '';
        switch (value) {
            case TaskStatus.Failed:
                type = 'failed';
                break;
            case TaskStatus.Cancelled:
                type = 'failed';
                break;
            case TaskStatus.Init:
                type = 'deployed';
                break;
            case  TaskStatus.Waiting:
                type = 'deployed';
                break;
            case  TaskStatus.Success:
                type = 'succeeded';
                break;
            case TaskStatus.Running:
            case TaskStatus.Canceling:
                type = 'running';
                break;
            default:
                break;
        }
        return type;
    }
}
