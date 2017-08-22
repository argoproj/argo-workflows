import * as moment from 'moment';
import { Component, Input } from '@angular/core';

import { Commit, Task } from '../../../model';
import { TaskService, CommitsService } from '../../../services';

@Component({
    selector: 'ax-recent-commits',
    templateUrl: './recent-commits.html',
    styles: [ require('./recent-commits.scss') ],
})
export class RecentCommitsComponent {

    @Input()
    public repoName: string;

    private commits: Commit[];
    private dataLoaded: boolean = false;

    constructor(private taskService: TaskService,
                private commitsService: CommitsService) {
    }

    public loadRecentCommits(task: Task) {
        let commitEndTime = moment.unix(task.commit.date);
        return this.commitsService.getCommitsAsync({maxTime: commitEndTime, limit: 5, branch: task.commit.branch, repo: task.commit.repo }, true).toPromise()
                .then(commits => commits.data).then(commits => {
            if (!commits.find(item => item.revision === task.commit.revision)) {
                commits = [task.commit].concat(commits);
            }
            this.dataLoaded = true;
            this.commits = commits;
        });
    }

    public clearRecentCommits() {
        this.commits = [];
        this.dataLoaded = false;
    }
}
