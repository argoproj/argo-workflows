import { Pipe, PipeTransform } from '@angular/core';
import { DeletedStatus } from '../model';

@Pipe({
    name: 'deletedStatus'
})
export class DeletedStatusPipe implements PipeTransform {
    transform(value: number, args: any[]) {
        let status = '';

        switch (value) {
            case DeletedStatus.Available:
                status = 'Available';
                break;
            case DeletedStatus.Expired:
                status = 'Permanently expired based on retention policy';
                break;
            case DeletedStatus.TemporaryDeleted:
                status = 'Temporarily deleted by a user';
                break;
            case DeletedStatus.PermanentlyDeleted:
                status = 'Permanently deleted by a user';
                break;
        }

        return status;
    }
}
