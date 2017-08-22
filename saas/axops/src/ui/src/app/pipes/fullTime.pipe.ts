import * as moment from 'moment';
import {Pipe, PipeTransform} from '@angular/core';

@Pipe({
    name: 'axFullTime'
})

export class FullTimePipe implements PipeTransform {
    transform(value: number, args: any[]) {
        return value ? moment.unix(value).format('H:mm:ss') : '';
    }
}
