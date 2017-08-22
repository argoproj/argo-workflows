import { Pipe, Injectable, PipeTransform } from '@angular/core';

@Pipe({
    name: 'axFilterBy'
})
@Injectable()
export class FilterByPipe implements PipeTransform {
    transform(items: string[], searchItem: string, propertyName: string): any {
        if (!items || items.length === 0 || !searchItem) {
            return items;
        }

        return items.filter((item: string) => {
            if (item.hasOwnProperty(propertyName)) {
                return item[propertyName].toLowerCase().indexOf(searchItem.toLowerCase()) !== -1;
            }
        });
    }
}
