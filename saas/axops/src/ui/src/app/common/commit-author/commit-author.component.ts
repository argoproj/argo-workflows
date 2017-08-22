import { Component, Input } from '@angular/core';

import { Commit } from '../../model';

@Component({
    selector: 'ax-commit-author',
    templateUrl: './commit-author.html',
    styles: [ require('./commit-author.scss') ],
})
export class CommitAuthorComponent {
    @Input()
    public set commit(value: Commit) {
        if (value && value.author) {
            this.userName = value.author.substring(0, value.author.lastIndexOf('<') - 1);
            this.userEmail = value.author.substring(value.author.lastIndexOf('<') + 1, value.author.trim().length - 1);
        }
        this.localCommit = value;
    };

    @Input('separate-line-position')
    public separateLinePosition: 'none'|'left'|'bottom'|'right'|'top' = 'none';

    public userName: string = '';
    public userEmail: string = '';
    public localCommit: Commit = new Commit();
}
