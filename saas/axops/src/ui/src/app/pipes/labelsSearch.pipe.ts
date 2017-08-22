import { Pipe, Injectable, PipeTransform } from '@angular/core';
import { Label } from '../model';

@Pipe({
    name: 'axLabelsSearch'
})
@Injectable()
export class LabelsSearchPipe implements PipeTransform {
    transform(items: Label[], searchKey: string): any {
        if (!items || items.length === 0 || !searchKey) {
            return items;
        }

        return items.filter((item: Label) => item.key.indexOf(searchKey) !== -1 ||
            item.value.indexOf(searchKey) !== -1 ||
            `${item.key}:${item.value}`.indexOf(searchKey) !== -1 );
    }
}
