import { Pipe, PipeTransform } from '@angular/core';

@Pipe({
  name: 'truncateTo'
})

export class TruncateToPipe implements PipeTransform {

  public transform(value = '', lettersNumber = 100) {
    const maxLength = lettersNumber;
    let ret = value;
    if (ret.length > maxLength) {
      ret = ret.substr(0, maxLength - 3) + '…';
    }
    return ret;
  }
}
