import {Component, Input} from '@angular/core';
import {Task} from '../../../../model';


@Component({
    selector: 'ax-jobs-details-box',
    templateUrl: './jobs-details-box.html',
})

export class JobsDetailsBoxComponent {
    @Input()
    tasks: Task[];
    @Input()
    canLoadMore: boolean;
    @Input()
    dataLoaded: boolean;
}
