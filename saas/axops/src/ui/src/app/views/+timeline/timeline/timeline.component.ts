import { Location } from '@angular/common';
import { Component, OnInit, OnDestroy } from '@angular/core';
import { Router, ActivatedRoute } from '@angular/router';
import { ReplaySubject, Subscription } from 'rxjs';

import { DateRange, DropdownMenuSettings } from 'argo-ui-lib/src/components';
import { HasLayoutSettings, LayoutSettings } from '../../layout';
import { ViewPreferencesService, AuthenticationService } from '../../../services';
import { Branch, ViewPreferences } from '../../../model';
import { ViewUtils, GLOBAL_SEARCH_TABS, GlobalSearchSetting } from '../../../common';
import { GlobalSearchFilters } from '../../../common/global-search-input/view-models';

import { JobFilter } from '../branches.view-models';

@Component({
    selector: 'ax-timeline',
    templateUrl: './timeline.html',
    styles: [ require('./timeline.scss') ],
})
export class TimelineComponent implements HasLayoutSettings, LayoutSettings, OnInit, OnDestroy {

    public jobFilter: JobFilter = new JobFilter();
    public selectedRepo: string = null;
    public selectedBranch: string = null;
    public currentView: string = 'overview';
    public showMyOnly: boolean = false;
    public branchesContext: Branch[] = [];
    public globalSearch: ReplaySubject<GlobalSearchSetting> = new ReplaySubject<GlobalSearchSetting>();
    public hasTabs: boolean = true;

    private subscriptions: Subscription[] = [];
    private viewPreferences: ViewPreferences;

    constructor(
        private router: Router,
        private location: Location,
        private route: ActivatedRoute,
        private authenticationService: AuthenticationService,
        private viewPreferencesService: ViewPreferencesService) {
    }

    public async ngOnInit() {
        this.viewPreferences = await this.viewPreferencesService.getViewPreferences();
        this.route.params.subscribe(params => {
            this.layoutDateRange.data = DateRange.fromRouteParams(params, -1);
            [this.selectedRepo, this.selectedBranch] = ViewUtils.getSelectedRepoBranch(params, this.viewPreferences);
            this.currentView = params['view'] || 'commit';
            this.showMyOnly = params['showMyOnly'] === 'true';
            this.jobFilter = new JobFilter();

            // hide "all" option in date range if it's a "overview" tab
            this.layoutDateRange.isAllDates = this.currentView !== 'overview';
            // "overview" tab doesn't have "all" date range, so if we navigate to this tab and the time range is set to "all", we need to change it to "today"
            if (this.currentView === 'overview' && this.layoutDateRange.data.isAllDates) {
                this.layoutDateRange.data = DateRange.fromRouteParams(DateRange.today().toRouteParams(), -1);
            }

            this.updateFiltersByView(this.currentView);
            this.toolbarFilters.model = [];

            if (this.showMyOnly ) {
                this.toolbarFilters.model.push('showMyOnly');
            }
            for (let status of Object.keys(this.jobFilter)) {
                if (params.hasOwnProperty(status)) {
                    this.jobFilter[status] = params[status] === 'true';
                    if (this.jobFilter[status]) {
                        this.toolbarFilters.model.push(status);
                    }
                }
            }
            this.globalSearch.next({
                suppressBackRoute: false,
                keepOpen: false,
                searchCategory: this.getCategoryByView(this.currentView),
            });

            this.branchesContext = this.viewPreferences.filterState.branches === 'my' ? this.viewPreferences.favouriteBranches : null;
            this.viewPreferencesService.updateViewPreferences(v => Object.assign(v.filterState, { selectedBranch: this.selectedBranch, selectedRepo: this.selectedRepo }));
        });
        this.subscriptions.push(this.viewPreferencesService.getViewPreferencesObservable().subscribe(viewPreferences => {
            this.branchesContext = viewPreferences.filterState.branches === 'my' ? viewPreferences.favouriteBranches : null;
            this.viewPreferences = viewPreferences;
        }));
    }

    public ngOnDestroy() {
        this.subscriptions.forEach(subcsribe => subcsribe.unsubscribe());
    }

    get pageTitle(): string {
        return 'Timeline';
    };

    get pageTitleIcon(): string {
        return this.selectedBranch ? 'branch' : null;
    }

    get breadcrumb(): { title: string, routerLink: any[] }[] {
        return ViewUtils.getBranchBreadcrumb(this.selectedRepo, this.selectedBranch, { url: '/app/timeline', params: { view: this.currentView } }, this.viewPreferences);
    }

    public branchNavPanelUrl = '/app/timeline';

    get currentViewTitle(): string {
        switch (this.currentView) {
            case 'job':
                return 'jobs';
            case 'overview':
                return 'branches';
            case 'commit':
                return 'commits';
        }
        return '';
    }

    get usernameFilter(): string {
        return this.showMyOnly ? this.authenticationService.getUsername() : null;
    }

    get globalAddActionMenu(): DropdownMenuSettings {
        if (this.currentView !== 'job') {
            return null;
        }

        return new DropdownMenuSettings([{
            title: 'Bulk Actions',
            iconName: '',
            action: async () => {
                this.navigateToBulkAction();
            },
        }], 'fa-ellipsis-v');
    }

    get layoutSettings(): LayoutSettings {
        return this;
    }

    public layoutDateRange = {
        data: DateRange.today(),
        onApplySelection: (date) => {
            this.router.navigate(['/app/timeline', this.getRouteParams(date.toRouteParams())]);
        },
        isAllDates: false
    };

    public toolbarFilters = {
        data: [],
        model: [],
        onChange: (data) => {
            for (let status of Object.keys(this.jobFilter)) {
                this.jobFilter[status] = data.indexOf(status) > -1;
            }
            this.showMyOnly = data.indexOf('showMyOnly') > -1;
            this.router.navigate(['/app/timeline', this.getRouteParams()]);
        }
    };

    public changeView(view: string) {
        this.toolbarFilters.data = [];
        this.router.navigate(['/app/timeline', this.getRouteParams({ view })]);
    }

    private navigateToBulkAction() {
        let filters = new GlobalSearchFilters();
        if (this.selectedRepo && !this.selectedBranch) {
            filters.jobs.repo = [this.selectedRepo];
        }
        if (this.selectedBranch ) {
            filters.jobs.branch = [`${this.selectedRepo} ${this.selectedBranch}`];
        }
        this.router.navigate(['/app/search', { category: 'jobs', backRoute: encodeURIComponent(this.location.path()), filters: JSON.stringify(filters)}]);
    }

    private getRouteParams(updatedParams?) {
        let params = { view: this.currentView };
        if (this.selectedBranch) {
            params['branch'] = encodeURIComponent(this.selectedBranch);
        }

        if (this.selectedRepo) {
            params['repo'] = encodeURIComponent(this.selectedRepo);
        }

        if (this.layoutDateRange && this.layoutDateRange.data) {
            params['days'] = encodeURIComponent(this.layoutDateRange.data.durationDays.toString());
            params['date'] = encodeURIComponent(this.layoutDateRange.data.endDate.unix().toString());
        }

        for (let status of Object.keys(this.jobFilter)) {
            params[status] = this.jobFilter[status] ? 'true' : 'false';
        }
        params['showMyOnly'] = this.showMyOnly;

        return ViewUtils.sanitizeRouteParams(params, updatedParams);
    }

    private updateFiltersByView(view) {
        this.currentView = view;
        this.toolbarFilters.data.length = 0;

        if (view !== 'overview') {
            this.toolbarFilters.data.push({
                value: 'showMyOnly',
                name: 'My ' + this.currentViewTitle,
                hasSeparator: true,
                icon: { className: view === 'overview' ? 'ax-icon-fav-selected' : 'ax-icon-user' },
            });
        }

        if (view === 'job' || view === 'overview') {
            this.toolbarFilters.data.push({
                value: 'failed',
                name: 'Failed',
                icon: { color: 'fail' },
            }, {
                value: 'succeeded',
                name: 'Succeeded',
                icon: { color: 'success' },
            });
        }
        if (view === 'job') {
            this.toolbarFilters.data.push({
                value: 'running',
                name: 'In-Progress',
                icon: { color: 'running' },
            }, {
                value: 'delayed',
                name: 'Queued',
                icon: { color: 'queued' },
            });
        }
    }

    private getCategoryByView(view): string {
        switch (view) {
            case 'job':
                return GLOBAL_SEARCH_TABS.JOBS.name;
            case 'overview':
                return GLOBAL_SEARCH_TABS.JOBS.name;
            case 'commit':
                return GLOBAL_SEARCH_TABS.COMMITS.name;
        }
        return 'commit';
    }
}
