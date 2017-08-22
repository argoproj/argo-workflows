import {Pipe, PipeTransform} from '@angular/core';

@Pipe({
    name: 'shortRevision'
})

export class ShortRevisionPipe implements PipeTransform {
    transform(value: string, args?: any[]) {
        return value ? value.substring(0, 7) : value;
    }
}
