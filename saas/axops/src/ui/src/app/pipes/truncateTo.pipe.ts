import {Pipe, PipeTransform} from '@angular/core';

@Pipe({
    name: 'truncateTo'
})

export class TruncateToPipe implements PipeTransform {

    transform(value = '', lettersNumber = 100) {
        let maxLength = lettersNumber;
        let ret = value;
        if (ret.length > maxLength) {
            ret = ret.substr(0, maxLength - 3) + '…';
        }
        return ret;
    }
}
