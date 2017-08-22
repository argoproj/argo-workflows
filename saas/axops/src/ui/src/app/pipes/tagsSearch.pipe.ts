import { Pipe, Injectable, PipeTransform } from '@angular/core';

@Pipe({
    name: 'axTagsSearch'
})
@Injectable()
export class TagsSearchPipe implements PipeTransform {
    transform(items: string[], searchItem: string): any {
        if (!items || items.length === 0 || !searchItem) {
            return items;
        }

        return items.filter((item: string) => item.indexOf(searchItem) !== -1);
    }
}
