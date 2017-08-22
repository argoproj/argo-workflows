import {Pipe, PipeTransform} from '@angular/core';

@Pipe({
    name: 'toServiceStatus'
})

export class ToServiceStatusPipe implements PipeTransform {
    transform(value: number, args: any[]) {

        switch (value) {
            case -2:
                return 'Canceled';
            case -1:
                return 'Failed';
            case 0:
                return 'Success';
            case 1:
                return 'Waiting';
            case 2:
                return 'Running';
            case 255:
                return 'Init';
            default:
                return 'Wrong status';
        }
    }
}
