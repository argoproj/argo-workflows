import { Component, Input } from '@angular/core';

import { Commit } from '../../../../model';

@Component({
    selector: 'ax-commit-details-box',
    templateUrl: './commit-details-box.html',
})
export class CommitDetailsBoxComponent {
    @Input()
    commit: Commit;
}
