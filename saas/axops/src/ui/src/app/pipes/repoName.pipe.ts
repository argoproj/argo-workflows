import {Pipe, PipeTransform} from '@angular/core';

declare let Math: any;

@Pipe({
    name: 'repoName'
})

export class RepoNamePipe implements PipeTransform {
    transform(value: string, args?: any[]) {
        if (!value) {
            value = '';
        }
        let repository: string[] = value.split('/');

        return repository[repository.length - 1];
    }
}
