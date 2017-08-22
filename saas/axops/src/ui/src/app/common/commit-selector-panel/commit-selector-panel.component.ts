import * as _ from 'lodash';
import { Component, Output, Input, EventEmitter } from '@angular/core';
import { Observable } from 'rxjs';

import { Branch, Repository, Commit } from '../../model';
import { BranchService, RepoService, CommitsService } from '../../services';
import { BranchesSortPipe } from '../../pipes';

@Component({
    selector: 'ax-commit-selector-panel',
    templateUrl: './commit-selector-panel.component.html',
    styles: [ require('./commit-selector-panel.scss') ],
})
export class CommitSelectorPanelComponent {

    repositories: Repository[];

    branches: Branch[];

    commits: Commit[];

    @Input()
    loading: boolean;

    @Input()
    onlyBranch: boolean = false;

    @Output()
    public selectedCommitChanged = new EventEmitter<Commit>();

    @Output()
    public onBranchChange = new EventEmitter<Branch>();

    @Output()
    public onClose = new EventEmitter<any>();

    @Input()
    public selectedCommit: Commit;

    @Input()
    set isShown(val: boolean) {
        this.search = '';
        if (val) {
            this.loadReposAndBranches();
        }
    }

    public search: string;
    public selectedRepository: Repository;
    public selectedBranch: Branch;

    constructor(
        private repoService: RepoService,
        private branchService: BranchService,
        private commitsService: CommitsService) {
    }

    selectRepository(repository: Repository) {
        this.selectedRepository = repository;
    }

    selectBranch(branch: Branch) {
        if (this.onlyBranch) {
            this.onBranchChange.emit(branch);
            this.selectedBranch = null;
            this.selectedRepository = null;
            this.onClose.emit({});
        } else {
            this.selectedBranch = branch;
            this.loading = true;
            let params = {
                author: null,
                repo: this.selectedBranch.repo,
                revision: null,
                branch: this.selectedBranch.name,
                minTime: null,
                maxTime: null,
                search: null,
                limit: 10,
                offset: null,
                sort: null,
                repo_branch: null,
            };
            this.commitsService.getCommitsAsync(params, true).subscribe(success => {
                this.loading = false;
                this.commits = success.data || [];
            }, err => {
                this.loading = false;
            });
        }
    }

    selectCommit(commit: Commit) {
        this.selectedCommit = commit;
        this.selectedCommitChanged.emit(commit);
    }

    isMatch(branch: Branch) {
        if (!this.search || this.search.trim() === '') {
            return true;
        }
        let words = this.search.split(' ').map(item => item.trim().toLowerCase()).filter(item => item !== '');
        return words.filter(word => `${branch.repo} ${branch.name}`.indexOf(word) > -1).length === words.length;
    }

    trackByBranch(i: number, branch: Branch) {
        return branch.repo + branch.name;
    }

    trackByRepo(i: number, repo: Repository) {
        return repo.url;
    }

    trackByCommit(i: number, commit: Commit) {
        return commit.revision;
    }

    shouldShowBranch(branch: Branch) {
        if (this.search || this.selectedRepository) {
            return this.isMatch(branch)
                && (!this.selectedRepository || this.selectedRepository && branch.repo === this.selectedRepository.url);
        }
        return false;
    }

    loadReposAndBranches() {
        this.loading = true;
        Observable.combineLatest(
            this.repoService.getReposAsync(true).map(
                res => {
                    this.repositories = _.map(res.data, (repo: string) => {
                        let repoSplit = repo.split('/');

                        return {
                            name: repoSplit[repoSplit.length - 1],
                            url: repo
                        };
                    }).sort((a, b) => {
                        if (a.name.toLowerCase() < b.name.toLowerCase()) {
                            return -1;
                        }
                        if (a.name.toLowerCase() > b.name.toLowerCase()) {
                            return 1;
                        }
                        return 0;
                    });
                }
            ),
            this.branchService.getBranchesAsync({}, true).map(
                res => this.branches = new BranchesSortPipe().transform(res.data)
            )).subscribe(res => {
            this.loading = false;
        }, err => {
            this.branches = [];
        });
    }
}
