import { Component, OnDestroy, OnInit, AfterViewInit, ViewChild } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { ReplaySubject, Subscription } from 'rxjs';

import { GlobalSearchFilters, GlobalSearchSetting, ViewUtils } from '../../common';
import { HasLayoutSettings, LayoutSettings } from '../layout';

@Component({
    selector: 'ax-global-search-results',
    templateUrl: './global-search-results.html',
    styles: [ require('./global-search-results.scss') ],
})
export class GlobalSearchResultsComponent implements OnInit, AfterViewInit, OnDestroy, HasLayoutSettings, LayoutSettings {
    public category: 'jobs' | 'commits' | 'applications' | 'deployments' | 'templates' | string;
    public backRoute: string;
    public filters: GlobalSearchFilters;
    public isFiltersVisible: boolean = false;
    public isAnyFilterSelected: boolean = false;
    public appName: string;

    @ViewChild('actionsDropDown')
    public actionsDropDown;

    // Layout settings
    public hiddenToolbar: boolean = true;
    public searchString: string;
    public globalSearch: ReplaySubject<GlobalSearchSetting> = new ReplaySubject<GlobalSearchSetting>();

    private subscriptions: Subscription[] = [];

    constructor(private route: ActivatedRoute,
                private router: Router) {
    }

    public ngOnInit() {
        this.subscriptions.push(this.route.params.subscribe(params => {
            this.searchString = params['keyword'];
            this.category = params['category'];
            this.backRoute = params['backRoute'] ? decodeURIComponent(params['backRoute']) : null;
            this.appName = params['appName'];
            this.filters = params['filters'] ? JSON.parse(decodeURIComponent(params['filters'])) : new GlobalSearchFilters();
            this.globalSearch.next({
                suppressBackRoute: true,
                keepOpen: true,
                searchString: this.searchString,
                searchCategory: this.category,
                backRoute: this.backRoute,
                filters: this.filters,
            });

            this.checkIfAnyFilterSelected(this.category);
        }));
    }

    public ngAfterViewInit() {
        // if there is selected any filter parameter, open filter after reload page
        this.isFiltersVisible = JSON.stringify(this.filters[this.category]) !== JSON.stringify(new GlobalSearchFilters()[this.category]);
    }

    public ngOnDestroy() {
        this.subscriptions.forEach(subscription => subscription.unsubscribe());
        this.subscriptions = [];
    }

    get layoutSettings(): LayoutSettings {
        return this;
    }

    public tabChange(tabName: string) {
        this.category = tabName;
        this.navigate(this.searchString);
    }

    public onFilterChange(globalSearchFilters: GlobalSearchFilters) {
        this.filters = globalSearchFilters;
        this.navigate(this.searchString);
    }

    private getRouteParams(updatedParams?) {
        let params = {};
        if (this.category) {
            params['category'] = encodeURIComponent(this.category);
        }

        if (this.backRoute) {
            params['backRoute'] = this.backRoute;
        }

        if (this.filters) {
            params['filters'] = encodeURIComponent(JSON.stringify(this.filters));
        }

        return ViewUtils.sanitizeRouteParams(params, updatedParams);
    }

    private navigate(searchString): void {
        searchString ? this.router.navigate(['/app/search/', this.searchString, this.getRouteParams()]) :
            this.router.navigate(['/app/search/', this.getRouteParams()]);
    }

    private checkIfAnyFilterSelected(category: string) {
        this.isAnyFilterSelected = JSON.stringify(this.filters[category]) !== JSON.stringify(new GlobalSearchFilters()[category]);
    }
}
