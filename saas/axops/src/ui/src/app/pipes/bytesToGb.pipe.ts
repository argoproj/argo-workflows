import {Pipe, PipeTransform} from '@angular/core';

declare let Math: any;

@Pipe({
    name: 'bytesToGb'
})

export class BytesToGbPipe implements PipeTransform {

    transform(value: number, args: any[]) {
        function bytesToSize(bytes) {
            let sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB'];
            if (bytes === 0) {
                return '0 Byte';
            }
            let i = parseInt(Math.floor(Math.log(bytes) / Math.log(1024)), 10);
            return Math.round(bytes / Math.pow(1024, i), 2) + ' ' + sizes[i];
        }
        return bytesToSize(value);
    }
}
