import { Component, Input } from '@angular/core';

import { Commit } from '../../../model';

@Component({
    selector: 'ax-commit-box',
    templateUrl: './commit-box.html',
    styles: [ require('./commit-box.scss') ],
})
export class CommitBoxComponent {
    @Input()
    commit: Commit;

    public get commitHistoryLink(): string {
        return this.commit ? `/app/timeline/commits/${this.commit.revision}` : '';
    }

}
