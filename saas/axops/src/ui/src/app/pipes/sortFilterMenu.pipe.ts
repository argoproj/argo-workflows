import { Pipe, PipeTransform } from '@angular/core';

@Pipe({
    name: 'axSortFilterMenu'
})
export class SortFilterMenuPipe implements PipeTransform {
    transform(array: any[]): any[] {
        let data: any[][] = [[]];

        array.forEach((value, index) => {
            data[data.length - 1].push(value);

            if (array.length - 1 === index || array[index].hasSeparator) {
                data[data.length - 1] = this.sort(data[data.length - 1]);
            }

            if (array.length - 1 !== index && array[index].hasSeparator) {
                data.push([]);
            }
        });

        return data.reduce((a, b) => a.concat(b));
    }

    private sort(array: any[]) {
        return array.sort((a: any, b: any) => {
            if (a.name < b.name) {
                return -1;
            } else if (a.name > b.name) {
                return 1;
            } else {
                return 0;
            }
        });
    }
}
