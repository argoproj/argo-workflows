import * as moment from 'moment';
import {Pipe, PipeTransform} from '@angular/core';

@Pipe({
    name: 'humanizeTime'
})

export class HumanizeTimePipe implements PipeTransform {
    transform(value: number, args: any[]) {
        return value ? moment.duration(value).humanize() : value;
    }
}
