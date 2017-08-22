import {Pipe, PipeTransform} from '@angular/core';

@Pipe({
    name: 'floatToPercents'
})

export class FloatToPercentsPipe implements PipeTransform {

    transform(value: number, args: any[]): string {
        return value ? (value * 100).toFixed(0) + ' %' : '';
    }
}
