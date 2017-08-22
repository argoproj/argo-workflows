import { Component, Output, Input, EventEmitter } from '@angular/core';
import { Branch, Repository  } from '../../model';

@Component({
    selector: 'ax-branch-selector-panel',
    templateUrl: './branch-selector-panel.component.html',
    styles: [ require('./branch-selector-panel.scss') ],
})
export class BranchSelectorPanelComponent {

    @Input()
    repositories: Repository[];

    @Input()
    branches: Branch[];

    @Input()
    loading: boolean;

    @Output()
    public selectedChanged = new EventEmitter<Branch[]>();

    @Input()
    public selectedBranches: Branch[] = [];

    @Input()
    set isShown(val: boolean) {
        this.search = '';
        this.selectedRepository = null;
    }

    public search: string;
    public selectedRepository: Repository;

    selectRepository(repository: Repository) {
        this.selectedRepository = repository;
    }

    isAdded(branch: Branch) {
        return this.selectedBranches.filter((favouriteBranch: Branch) => {
                return favouriteBranch.name === branch.name && favouriteBranch.repo === branch.repo;
            }).length > 0;
    }

    isMatch(branch: Branch) {
        if (!this.search || this.search.trim() === '') {
            return true;
        }
        let words = this.search.split(' ').map(item => item.trim().toLowerCase()).filter(item => item !== '');
        return words.filter(word => `${branch.repo} ${branch.name}`.indexOf(word) > -1).length === words.length;
    }

    removeBranch(branch: Branch) {
        this.selectedBranches = this.selectedBranches.filter(
            (fBranch: Branch) => !(fBranch.name === branch.name && fBranch.repo === branch.repo)
        );
        this.selectedChanged.emit(this.selectedBranches);
    }

    addBranch(branch: Branch) {
        let newFavouriteBranch = this.branches.filter(
            (b: Branch) => b.name === branch.name && b.repo === branch.repo
        )[0];
        this.selectedBranches.push(newFavouriteBranch);

        this.selectedChanged.emit(this.selectedBranches);
    }

    trackByBranch(i: number, branch: Branch) {
        return branch.repo + branch.name;
    }

    trackByRepo(i: number, repo: Repository) {
        return repo.url;
    }

    shouldShowBranch(branch: Branch) {
        if (this.search || this.selectedRepository) {
            return this.isMatch(branch)
                && (!this.selectedRepository || this.selectedRepository && branch.repo === this.selectedRepository.url);
        }
        return false;
    }
}
