import { Pipe, Injectable, PipeTransform } from '@angular/core';

@Pipe({
    name: 'axBranchesSearch'
})
@Injectable()
export class BranchesSearchPipe implements PipeTransform {
    transform(items: any[], searchKey: string): any {
        if (!items || items.length === 0 || !searchKey) {
            return items;
        }

        return items.filter(item => item.name.indexOf(searchKey) !== -1 || item.repo.indexOf(searchKey) !== -1);
    }
}
