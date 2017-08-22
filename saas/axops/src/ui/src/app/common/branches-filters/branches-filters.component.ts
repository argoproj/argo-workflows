import * as _ from 'lodash';
import { Component, OnInit, Input } from '@angular/core';

import { RepoService, BranchService } from '../../services';
import { Branch } from '../../model';
import { SortOperations } from '../../common';

@Component({
    selector: 'ax-branches-filters',
    templateUrl: './branches-filters.html',
    styles: [ require('./branches-filters.scss') ],
})
export class BranchesFiltersComponent implements OnInit {

    @Input()
    selectedRepo: string = null;
    @Input()
    selectedBranch: string = null;

    allRepositories: {name: string, url: string}[];
    repositories: {name: string, url: string}[];
    allBranches: Branch[] = [];
    branches: Branch[] = [];
    selectedRepoName: string = '';
    searchString: string = '';

    public static parseRepoUrl(repo: string): { name: string, url: string } {
        let repoSplit = repo.split('/');
        return {
            name: repoSplit[repoSplit.length - 1],
            url: repo
        };
    }

    public static formatSelection(selectedRepo: string, selectedBranch: string, pageName: string = 'branches'): string {
        if (!selectedRepo && !selectedBranch) {
            return `All ${pageName}`;
        } else if (!selectedBranch) {
            let repoInfo = BranchesFiltersComponent.parseRepoUrl(selectedRepo);
            return `All ${pageName} of ${repoInfo.name}`;
        } else {
            let repoInfo = BranchesFiltersComponent.parseRepoUrl(selectedRepo);
            return `.../${repoInfo.name}/${selectedBranch}`;
        }
    }

    constructor(private repoService: RepoService, private branchService: BranchService) {}

    toggleAllRepos() {
        if (!this.selectAllRepos) {
            this.selectedRepo = null;
            this.selectedBranch = null;
        } else {
            this.selectedRepo = this.repositories[0].url;
        }
    }

    toggleAllBranches() {
        if (!this.selectAllBranches) {
            this.selectedBranch = null;
        } else {
            this.selectedBranch = this.branches.filter(branch => branch.repo === this.selectedRepo)[0].name;
        }
    }

    get selectAllRepos(): boolean {
        return this.selectedRepo == null;
    }

    get selectAllBranches(): boolean {
        return this.selectedBranch == null;
    }

    ngOnInit() {
        this.repoService.getReposAsync(true).subscribe(
            success => {
                this.allRepositories = _.map(success.data, (repo: string) => BranchesFiltersComponent.parseRepoUrl(repo)).sort((a, b) => {
                    if (a.name.toLowerCase() < b.name.toLowerCase()) {
                        return -1;
                    }
                    if (a.name.toLowerCase() > b.name.toLowerCase()) {
                        return 1;
                    }
                    return 0;
                });
                this.repositories = this.allRepositories;
            }
        );
        this.branchService.getBranchesAsync({}, true).subscribe(
            success => {
                this.allBranches = success.data;
                let masterBranchIndex = this.allBranches.findIndex(b => {
                    return b.name.toLowerCase() === 'master';
                });
                if (masterBranchIndex >= 0) {
                    let masterBranch = [this.allBranches[masterBranchIndex]];
                    this.allBranches.splice(masterBranchIndex, 1);
                    this.allBranches = _.union(masterBranch, SortOperations.sortBy(this.allBranches, 'name', true));
                } else {
                    this.allBranches = SortOperations.sortBy(this.allBranches, 'name', true);
                }
                this.branches = this.allBranches;
            }
        );

    }

    selectRepo(repo: {name: string, url: string}) {
        this.selectedRepoName = repo.name;
        this.selectedRepo = repo.url;
    }

    selectBranch(branch: string) {
        this.selectedBranch = branch;
    }

    search(input: string) {
        this.searchString = input;
        if (input.length > 0) {
            this.branches = this.allBranches.filter(b => {
                // selected branch always on the list
                return b.name.indexOf(input) >= 0 || b.name === this.selectedBranch;
            });
        } else {
            this.branches = this.allBranches;
        }
    }
}
