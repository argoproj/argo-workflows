import { Pipe, Injectable, PipeTransform } from '@angular/core';

@Pipe({
    name: 'axBranchesSort'
})
@Injectable()
export class BranchesSortPipe implements PipeTransform {
    transform(items: any[]): any {
        return items.sort((a, b) => {
            if (
                a.name.toLowerCase() === 'master'
                || (a.name.toLowerCase() < b.name.toLowerCase() && b.name.toLowerCase() !== 'master')
            ) {
                return -1;
            }

            if (a.name.toLowerCase() > b.name.toLowerCase() || b.name.toLowerCase() === 'master') {
                return 1;
            }

            return 0;
        });
    }
}
