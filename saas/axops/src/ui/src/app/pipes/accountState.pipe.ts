import {Pipe, PipeTransform} from '@angular/core';

import {AccountState} from '../model';

@Pipe({
    name: 'accountState'
})

export class AccountStatePipe implements PipeTransform {
    transform(value: number, args: any[]) {
        let state = '';

        switch (value) {
            case AccountState.Init:
                state = 'Confirmation Email Sent';
                break;
            case AccountState.Active:
                state = 'Active';
                break;
            case AccountState.Inactive:
                state = 'Inactive';
                break;
            case AccountState.Deleted:
                state = 'Disabled';
                break;
        }

        return state;
    }
}
