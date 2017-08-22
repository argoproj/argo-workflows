import {Pipe, PipeTransform} from '@angular/core';

@Pipe({
    name: 'centsToDollars'
})

export class CentsToDollarsPipe implements PipeTransform {
    transform(value: number, digitsAfterColon = 6) {
        return Number((value / 100).toFixed(digitsAfterColon)) === 0 ? '$0' : '$' + (value / 100).toFixed(digitsAfterColon);
    }
}
