import { Component, OnInit, Input, OnDestroy, ViewChild, AfterContentInit } from '@angular/core';
import { FormControl } from '@angular/forms';
import { Observable, Subscription } from 'rxjs';

import { Branch, ViewPreferences, Repository } from '../../model';
import { ViewPreferencesService, BranchService } from '../../services';
import { BranchesSortPipe } from '../../pipes/branches-sort.pipe';

enum BranchesTabs {
    Repositories,
    Favourites
}

@Component({
    selector: 'ax-branches-panel',
    templateUrl: './branches-panel.component.html',
    styles: [ require('./branches-panel.scss') ],
})
export class BranchesPanelComponent implements OnInit, OnDestroy, AfterContentInit {
    @Input()
    linkSegments: string = '/app/timeline';

    @Input()
    repositories: Repository[];

    @Input()
    loading: boolean;

    @Input()
    hideFavourites: boolean;

    @Input()
    set isShown(val: boolean) {
        this.currentTab = this.favouriteBranches && this.favouriteBranches.length > 0 ? BranchesTabs.Favourites : BranchesTabs.Repositories;
        this.selectedRepository = null;
    }

    public branches: Branch[];
    public currentTab: BranchesTabs = BranchesTabs.Favourites;
    public branchesTabs = BranchesTabs;
    public selectedRepository: Repository;
    public showHeader: boolean;

    public searchControl = new FormControl();
    private favouriteBranches: Branch[] = [];
    private subscriptions: Subscription[] = [];
    private branchesSubscription: Subscription;
    private term: string = '';

    @ViewChild('panelHeader')
    private panelHeader;

    get branchesToShow() {
        return this.currentTab === BranchesTabs.Favourites && !this.searchControl.value ? this.favouriteBranches : this.branches;
    }

    constructor(private viewPreferencesService: ViewPreferencesService, private branchService: BranchService) {
    }

    public ngOnInit() {
        this.subscriptions.push(this.viewPreferencesService.getViewPreferencesObservable().subscribe((v: ViewPreferences) => {
            this.favouriteBranches = v.favouriteBranches;
        }));

        this.subscriptions.push(this.searchControl.valueChanges
            .debounceTime(400)
            .subscribe(term => {
                if (term) {
                    term = `*${term}*`;
                }
                this.term = term;
                this.getBranches(term);
            })
        );
    }

    public ngAfterContentInit() {
        this.showHeader = this.panelHeader.nativeElement.children.length > 0;
    }

    public ngOnDestroy() {
        this.subscriptions.forEach(subscription => subscription.unsubscribe());
    }

    public selectTab(tab: BranchesTabs) {
        this.currentTab = tab;
        this.selectRepository(null);
    }

    public selectRepository(repository: Repository) {
        this.selectedRepository = repository;
        this.branches = null;
        this.loading = false;
        if (repository) {
            this.getBranches();
        } else {
            if (this.branchesSubscription) {
                this.branchesSubscription.unsubscribe();
            }
            this.searchControl.setValue(null);
        }
    }

    public isFavourite(branch: Branch) {
        return this.favouriteBranches.filter((favouriteBranch: Branch) => {
            return favouriteBranch.name === branch.name && favouriteBranch.repo === branch.repo;
        }).length > 0;
    }

    public removeBranchFromFavourite(branch: Branch) {
        this.favouriteBranches = this.favouriteBranches.filter(
            (fBranch: Branch) => !(fBranch.name === branch.name && fBranch.repo === branch.repo)
        );

        this.viewPreferencesService.updateViewPreferences(viewPreferences => viewPreferences.favouriteBranches = this.favouriteBranches);
    }

    public addBranchToFavourite(branch: Branch) {
        let newFavouriteBranch = this.branches.filter(
            (b: Branch) => b.name === branch.name && b.repo === branch.repo
        )[0];
        this.favouriteBranches.push(newFavouriteBranch);

        this.viewPreferencesService.updateViewPreferences(viewPreferences => viewPreferences.favouriteBranches = this.favouriteBranches);
    }

    public trackByBranch(i: number, branch: Branch) {
        return branch.repo + branch.name;
    }

    public trackByRepo(i: number, repo: Repository) {
        return repo.url;
    }

    private getBranches(search?: string) {
        let getBranchCalls: Observable<{ data: Branch[] }>[] = new Array();
        if (this.branchesSubscription) {
            this.branchesSubscription.unsubscribe();
        }
        this.branches = null;
        if (!this.selectedRepository && !search) {
            return;
        }
        this.loading = true;



        getBranchCalls.push(this.branchService.getBranchesAsync(
            {
                limit: 10,
                repo: this.selectedRepository && this.selectedRepository.url,
                branch: search, orderBy: this.selectedRepository ? '-commit_date' : ''
            }, true
        ));
        if (!search) {
            getBranchCalls.push(this.branchService.getBranchesAsync(
                { repo: this.selectedRepository && this.selectedRepository.url, branch: 'master' }, true
            ));
        }
        this.branchesSubscription = Observable.forkJoin(getBranchCalls).subscribe(success => {
            let branches = success[0].data;
            if (success.length === 2) {
                if (branches.find(item => item.name === 'master') === undefined) {
                    let master = success[1].data.find(item => item.name === 'master');
                    if (master) {
                        branches = branches.concat(master);
                    }
                }
            }

            this.branches = new BranchesSortPipe().transform(branches);
            this.loading = false;
        });
    }
}
