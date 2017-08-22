import {Pipe, PipeTransform} from '@angular/core';

@Pipe({
    name: 'millisecondsToSeconds'
})

export class MillisecondsToSecondsPipe implements PipeTransform {
    transform(value: number, args: any[]): any {
        return value ? value / 1000 : value;
    }
}
