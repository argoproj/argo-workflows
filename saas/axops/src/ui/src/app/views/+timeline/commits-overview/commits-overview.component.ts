import * as moment from 'moment';
import { Component, OnChanges, Input } from '@angular/core';
import { Subscription, Observable } from 'rxjs';

import { Commit, Template, Branch } from '../../../model';
import { CommitsService } from '../../../services';
import { DateRange } from 'argo-ui-lib/src/components';
import { LaunchPanelService } from '../../../common/multiple-service-launch-panel/launch-panel.service';

@Component({
    selector: 'ax-commits-overview',
    templateUrl: './commits-overview.html',
    styles: [ require('./commits-overview.scss') ],
})
export class CommitsOverviewComponent implements OnChanges {
    protected readonly bufferSize: number = 20;

    @Input()
    public dateRange: DateRange;
    @Input()
    public selectedRepo: string = null;
    @Input()
    public selectedBranch: string = null;
    @Input()
    public selectedUsername: string = null;
    @Input()
    public searchString: string = null;
    @Input()
    public branchesContext: Branch[];

    public templates: Template[] = [];
    public expandedCommit: string;
    public mockupList: any[] = [];

    private commits: Commit[];

    private maxTime: number;
    private canScroll: boolean = false;
    private dataLoaded: boolean = false;
    private getCommitsSubscription: Subscription;

    constructor(private commitsService: CommitsService, private launchPanelService: LaunchPanelService) {
    }

    public ngOnChanges() {
        this.getCommitsUnsubscribe();
        this.mockupList = [];
        this.commits = null;
        // count how many loader template elements we need to fill full screen
        for (let i = 0; i < window.innerHeight / 150; i++) { // 150 is height of single element
            this.mockupList.push('');
        }
        this.getCommitsSubscription = this.getCommitsAsync(0, this.bufferSize, null, true).subscribe(res => {
            this.commits = res.data || [];
            this.setScrollAtributes(this.commits.length);
            this.dataLoaded = true;
        }, err => {
            this.dataLoaded = true;
        });
    }

    public openServiceTemplatePanel(commit: Commit): void {
        this.launchPanelService.openPanel(commit);
    }

    public onScroll(): void {
        if (this.canScroll) {
            this.dataLoaded = false;
            this.canScroll = false;
            this.getCommitsAsync(null, this.bufferSize, moment.unix(this.maxTime), true)
                .subscribe(success => {
                    this.dataLoaded = true;
                    this.commits = this.commits.concat(success.data || []);

                    this.setScrollAtributes(success.data.length);
                }, err => {
                    this.dataLoaded = true;
                });
        }
    }

    public selectCommit(revision: string): void {
        this.expandedCommit = this.expandedCommit === revision ? null : revision;
    }

    private setScrollAtributes(takenCommitsLength: number): void {
        if (takenCommitsLength === this.bufferSize) {
            this.canScroll = true;
            // I have to minus 1 to date because last element from earlier query is the same as first element in
            // next if we set maxtime as last element date
            this.maxTime = this.commits[this.commits.length - 1].date - 1;
        }
    }

    private getCommitsAsync(skip?: number, limit?: number, lazyLoadingMaxTime?: moment.Moment, hideLoader?: boolean): Observable<{data: Commit[]}> {
        let repoBranch = null;
        if (this.branchesContext && !this.selectedBranch && !this.selectedRepo) {
            repoBranch = {};
            this.branchesContext.forEach(item => {
                let branches = (repoBranch[item.repo] || []);
                branches.push(item.name);
                repoBranch[item.repo] = branches;
            });
        }

        let params = {
            repo: this.selectedRepo,
            branch: this.selectedBranch,
            author: this.selectedUsername,
            minTime: null,
            maxTime: null,
            search: null,
            limit: null,
            offset: null,
            sort: null,
            repo_branch: repoBranch,
        };
        if (!this.dateRange.isAllDates) {
            params.minTime = this.dateRange.startDate;
            params.maxTime = this.dateRange.endDate;
        }
        if (limit) {
            params.limit = limit;
        }
        if (skip) {
            params.offset = skip;
        }
        if (lazyLoadingMaxTime) {
            params.maxTime = lazyLoadingMaxTime;
        }
        params.search = this.searchString;
        // default sort order is date ascending
        // Removed due to a bug
        // params.sort = '-date';
        return this.commitsService.getCommitsAsync(params, hideLoader);
    }

    private getCommitsUnsubscribe() {
        if (this.getCommitsSubscription) {
            this.getCommitsSubscription.unsubscribe();
            this.getCommitsSubscription = null;
        }
    }
}
