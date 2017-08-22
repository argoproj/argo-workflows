import { Component, Input } from '@angular/core';

import { Commit } from '../../model';

@Component({
    selector: 'ax-commit-description',
    templateUrl: './commit-description.html',
    styles: [ require('./commit-description.scss') ],
})
export class CommitDescriptionComponent {
    @Input('commit-description-details')
    public commitDescriptionDetails: { size: number, isExpanded: boolean } = { size: 300, isExpanded: false };

    @Input()
    public commit: Commit = new Commit();

    @Input()
    public showBranchAndRepo: boolean = true;
    @Input()
    public showCommitId: boolean = true;

    commitDescriptionToggle() {
        this.commitDescriptionDetails.isExpanded = !this.commitDescriptionDetails.isExpanded;
        this.commitDescriptionDetails.size = this.commitDescriptionDetails.isExpanded ? this.commit.description.length : 300;
    }
}
