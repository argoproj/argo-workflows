import {Pipe, PipeTransform} from '@angular/core';

@Pipe({
    name: 'capitalizeSourceName'
})

export class CapitalizeSourceNamePipe implements PipeTransform {

    transform(value: string, args: any[]) {
        function capitzalieSourceName(input) {
            let sourceName = input;

            switch (input) {
                case 'bitbucket':
                    sourceName = 'Bitbucket';
                    break;
                case 'github':
                    sourceName = 'GitHub';
                    break;
                case 'git':
                    sourceName = 'Git';
                    break;
                case 'codecommit':
                    sourceName = 'CodeCommit';
                    break;
            }

            return sourceName;
        }
        return capitzalieSourceName(value);
    }
}
