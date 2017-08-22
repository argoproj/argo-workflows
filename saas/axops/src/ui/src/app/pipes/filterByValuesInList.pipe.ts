import { Pipe, Injectable, PipeTransform } from '@angular/core';

@Pipe({
    name: 'axFilterByValuesInList'
})
@Injectable()
export class FilterByValuesInListPipe implements PipeTransform {
    transform(items: string[], searchItems: any, propertyName: string): any {
        if (!items || items.length === 0 || !searchItems || searchItems.length === 0) {
            return items;
        }

        return items.filter((item: string) => {
            if (item.hasOwnProperty(propertyName)) {
                return item[propertyName].some(v => searchItems.includes(v));
            }
        });
    }
}
