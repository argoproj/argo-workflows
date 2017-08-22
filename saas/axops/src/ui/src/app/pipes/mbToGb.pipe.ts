import {Pipe, PipeTransform} from '@angular/core';

declare let Math: any;

@Pipe({
    name: 'mbToGb'
})

export class MbToGbPipe implements PipeTransform {

    transform(value: number, args: any[]) {
        if (!value) {
            value = 0;
        }

        function megaBytesToSize(mb) {
            let sizes = ['MB', 'GB', 'TB'];
            if (mb === 0) {
                return '0 MB';
            }
            let i = parseInt(Math.floor(Math.log(mb) / Math.log(1024)), 10);
            return Math.round(mb / Math.pow(1024, i), 2) + ' ' + sizes[i];
        }
        return megaBytesToSize(value);
    }
}
