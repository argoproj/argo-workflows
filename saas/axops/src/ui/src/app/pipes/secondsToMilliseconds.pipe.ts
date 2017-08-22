import {Pipe, PipeTransform} from '@angular/core';

@Pipe({
    name: 'secondsToMilliseconds'
})

export class SecondsToMillisecondsPipe implements PipeTransform {
    transform(value: number, args: any[]): any {
        return value ? value * 1000 : value;
    }
}
