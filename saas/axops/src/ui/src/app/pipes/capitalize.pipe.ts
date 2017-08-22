import {Pipe, PipeTransform} from '@angular/core';

@Pipe({
    name: 'capitalize'
})

export class CapitalizePipe implements PipeTransform {

    transform(value: string, args: any[]) {
        function capitalize(input) {
            return (!!input) ? input.charAt(0).toUpperCase() + input.substr(1).toLowerCase() : '';
        }
        return capitalize(value);
    }
}
