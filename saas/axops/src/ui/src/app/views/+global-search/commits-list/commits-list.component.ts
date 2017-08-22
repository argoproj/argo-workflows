import * as moment from 'moment';
import { Component, Input, OnChanges, OnDestroy, ViewChild } from '@angular/core';
import { Observable, Subscription } from 'rxjs';

import { TimeRangePagination, TimerangePaginationComponent, CommitsFilters } from '../../../common';
import { Commit, CommitFieldNames } from '../../../model';
import { CommitsService, GlobalSearchService } from '../../../services';

@Component({
    selector: 'ax-commits-list',
    templateUrl: './commits-list.html',
})
export class CommitsListComponent implements OnChanges, OnDestroy {
    protected limit: number = 10;

    @Input()
    public filters: CommitsFilters;

    @Input()
    public searchString: string;

    @ViewChild('timerangePagination')
    timerangePaginationComponent: TimerangePaginationComponent;

    public commits: Commit[] = [];
    public maxTime: number = 0;
    public params: CommitsFilters;
    public dataLoaded: boolean = false;
    public pagination: TimeRangePagination = {
        limit: this.limit,
        listLength: 10,
        maxTime: 0
    };

    private subscriptions: Subscription[] = [];

    constructor(private commitsService: CommitsService, private globalSearchService: GlobalSearchService) {
    }

    public ngOnChanges() {
        // clean pagination if filter was changed
        this.timerangePaginationComponent.cleanPagination();

        // need to map readable statuses string representation to numbers
        this.params = {
            branch: this.filters.branch,
            repo: this.filters.repo,
            authors: this.filters.authors,
            committers: this.filters.committers,
        };
        // restart pagination if changed search parameters
        this.pagination = {limit: this.limit, listLength: this.commits.length, maxTime: this.maxTime};
        this.updateCommits(this.params, this.pagination, true);
    }

    public ngOnDestroy() {
        this.subscriptions.forEach(subscription => subscription.unsubscribe());
        this.subscriptions = [];
    }

    public onPaginationChange(pagination: TimeRangePagination) {
        this.limit = pagination.limit;
        this.timerangePaginationComponent.cleanPagination();
        this.updateCommits(this.params, pagination, true);
    }

    public navigateToDetails(revision: string): void {
        this.globalSearchService.navigate(['/app/timeline/commits/', revision]);
    }

    private updateCommits(params: CommitsFilters, pagination: TimeRangePagination, hideLoader?: boolean) {
        this.dataLoaded = false;
        this.pagination.limit += 1;

        this.subscriptions.push(this.getCommitsAsync(params, pagination, hideLoader).subscribe(result => {
            this.dataLoaded = true;

            this.commits = result.data.slice(0, this.limit);

            this.pagination = {
                limit: this.limit,
                maxTime: result.data.length > 2 ? result.data[result.data.length - 2].date - 1 : null,
                listLength: this.commits.length,
                hasMore: result.data.length > this.limit
            };
        }, error => {
            this.dataLoaded = true;
            this.commits = [];
        }));
    }

    private getCommitsAsync(params: CommitsFilters, pagination: TimeRangePagination, hideLoader?: boolean):
        Observable<{data: Commit[]}> {
        let parameters = {
            repo: null,
            branch: null,
            author: null,
            committer: null,
            minTime: null,
            maxTime: null,
            search: this.searchString,
            limit: null,
            offset: null,
            searchFields: [
                CommitFieldNames.description,
                CommitFieldNames.author,
                CommitFieldNames.committer,
                CommitFieldNames.repo,
                CommitFieldNames.branch,
            ],
            sort: null,
            repo_branch: null
        };

        if (pagination.limit) {
            parameters.limit = pagination.limit;
        }

        if (pagination.maxTime) {
            parameters.maxTime = moment.unix(pagination.maxTime);
        }

        if (params.authors && params.authors.length) {
            parameters.author = params.authors;
        }

        if (params.committers && params.committers.length) {
            parameters.committer = params.committers;
        }

        // filtering by repo is possible only for single repo
        if (params.repo && params.repo.length === 1 && params.branch && params.branch.length === 0) {
            parameters.repo = params.repo;
        }

        // for filtering by multiple repo or by branch (also by multiple branch) you have to use repo_branch
        if (params.repo && params.repo.length > 1 || params.repo && params.branch && params.branch.length) {
            let repo_branch = {};
            let branchesList = params.branch.map(i => {
                return {repo: i.split(' ')[0], branch: i.split(' ')[1]};
            });

            if (params.repo.length > 1) {
                params.repo.forEach(repo => {
                    repo_branch[repo] = [];
                });
            }

            branchesList.forEach(item => {
                if (repo_branch.hasOwnProperty(item.repo)) {
                    repo_branch[item.repo].push(item.branch);
                } else {
                    repo_branch[item.repo] = [item.branch];
                }
            });

            parameters.repo_branch = repo_branch;
        }

        // default sort order is date ascending
        // Removed due to a bug
        // params.sort = '-date';
        return this.commitsService.getCommitsAsync(parameters, hideLoader);
    }
}
