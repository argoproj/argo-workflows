import * as _ from 'lodash';
import {
    Component, Output, Input, EventEmitter, ElementRef, OnInit, HostListener,
    SimpleChanges, OnChanges, OnDestroy
} from '@angular/core';
import { Location } from '@angular/common';
import { FormControl, FormGroup } from '@angular/forms';
import { Router } from '@angular/router';
import { Observable, Subject } from 'rxjs';

import { ViewUtils } from '../view-utils';
import { SearchIndex, SEARCH_INDEX_TYPE } from '../../model';
import { GlobalSearchService, FeaturesSetsService } from '../../services';
import {
    GLOBAL_SEARCH_TABS, GLOBAL_SEARCH_SUGGESTION_FIELDS_CONFIG, GlobalSearchFilters, GlobalSearchSetting,
    SearchHistory, SearchHistoryItem, LOCAL_SEARCH_CATEGORIES
} from './view-models';

export class GlobalSearchParams {
    searchCategory: string = '';
    searchString: string = '';
}

const NAVIGATION_SHORTCUTS = {
    TIMELINE: {
        name: 'timeline',
        icon: 'ax-icon-timeline',
        url: '/app/timeline',
    },
    APPLICATION: {
        name: 'application',
        icon: 'ax-icon-application',
        url: '/app/applications',
        featureSets: ['full', 'limited_aws'],
    },
    TEMPLATES: {
        name: 'templates',
        icon: 'ax-icon-template',
        url: '/app/service-catalog/overview',
        featureSets: ['full', 'limited_aws', 'limited'],
    },
    POLICIES: {
        name: 'policies',
        icon: 'ax-icon-policies',
        url: '/app/policies/overview',
        featureSets: ['full', 'limited_aws', 'limited'],
    },
    APPSTORE: {
        name: 'appstore',
        icon: 'ax-icon-appstore',
        url: '/app/ax-catalog',
        featureSets: ['full', 'limited_aws', 'limited'],
    },
    CASHBOARD: {
        name: 'cashboard',
        icon: 'ax-icon-cashboard',
        url: '/app/cashboard',
        featureSets: ['full'],
    },
    METRICS: {
        name: 'metrics',
        icon: 'ax-icon-metrics',
        url: '/app/metrics',
        featureSets: ['full', 'limited_aws', 'limited'],
    },
    FIXTURES: {
        name: 'fixtures',
        icon: 'ax-icon-fixture',
        url: '/app/fixtures',
        featureSets: ['full', 'limited_aws'],
    },
    VOLUMES: {
        name: 'volumes',
        icon: 'ax-icon-volume',
        url: '/app/volumes',
        featureSets: ['full', 'limited_aws'],
    },
    HOSTS: {
        name: 'hosts',
        icon: 'ax-icon-axcluster',
        url: '/app/hosts',
        featureSets: ['full', 'limited_aws'],
    }
};

interface Suggestion {
    type: string;
    term?: string;
    details: any;
    icon?: string;
}

interface SearchTerm {
    term: string;
    category: string;
}

@Component({
    selector: 'ax-global-search-input',
    templateUrl: './global-search-input.html',
    styles: [require('./global-search-input.scss')],
})
export class GlobalSearchInputComponent implements OnInit, OnChanges, OnDestroy {
    @Input()
    limit: number = 7;
    @Input()
    settings: GlobalSearchSetting;

    @Output()
    update: EventEmitter<GlobalSearchParams> = new EventEmitter<GlobalSearchParams>();
    @Output()
    onSelect = new EventEmitter();

    public loadingSuggestion: boolean = false;
    public isSearchInputExpanded: boolean;
    public isDropdownVisible: boolean = false;
    public backRoute: string;
    public searchForm: FormGroup;
    public dropdownList: Suggestion[] = [];

    private selectedItemIndex: number = -1;
    private currentStoreVersion: number = 4; // to avoid possible conflicts with data storaged in local storage, increase number after change in the filter format
    private currentStoreKey: string = '';
    private searchHistoryList: SearchHistory = new SearchHistory();
    private searchTerms = new Subject<SearchTerm>();
    private dropdownNavigationShortcuts: { name: string, icon: string, url: string, featureSets?: string[] }[] = [
        NAVIGATION_SHORTCUTS.TIMELINE,
        NAVIGATION_SHORTCUTS.APPLICATION,
        NAVIGATION_SHORTCUTS.TEMPLATES,
        NAVIGATION_SHORTCUTS.POLICIES,
        NAVIGATION_SHORTCUTS.APPSTORE,
        NAVIGATION_SHORTCUTS.CASHBOARD,
        NAVIGATION_SHORTCUTS.METRICS,
        NAVIGATION_SHORTCUTS.FIXTURES,
        NAVIGATION_SHORTCUTS.VOLUMES,
        NAVIGATION_SHORTCUTS.HOSTS,
    ];

    private searchInCategories: any[] = [
        GLOBAL_SEARCH_TABS.COMMITS,
        GLOBAL_SEARCH_TABS.JOBS,
        GLOBAL_SEARCH_TABS.TEMPLATES,
        GLOBAL_SEARCH_TABS.APPLICATIONS,
        GLOBAL_SEARCH_TABS.DEPLOYMENTS,
    ];

    constructor(private elementRef: ElementRef,
                private location: Location,
                private router: Router,
                private globalSearchService: GlobalSearchService,
                private featuresSetsService: FeaturesSetsService) {
        this.featuresSetsService.getFeaturesSet().then(featureSet => {
            this.dropdownNavigationShortcuts = this.dropdownNavigationShortcuts.filter(item => !item.featureSets || item.featureSets.indexOf(featureSet) > -1);
            this.searchInCategories = this.searchInCategories.filter(item => !item.featureSets || item.featureSets.indexOf(featureSet) > -1);
        });
    }

    public ngOnInit() {
        this.currentStoreKey = 'ax-global-search-history';
        this.initSearchHistoryList();
        this.searchForm = new FormGroup({
            inputControl: new FormControl(this.settings.searchString),
        });

        // search for suggestions
        this.searchTerms
            .debounceTime(300)
            .distinctUntilChanged((a: SearchTerm, b: SearchTerm) => {
                return a.term === b.term && a.category === b.category;
            })
            .switchMap(term => {
                let type = this.getTypeByCategory(term.category);
                if (term.term && type && !this.settings.hideSearchHistoryAndSuggestions) {
                    this.loadingSuggestion = true;

                    return this.globalSearchService.getSuggestions({
                        type: type,
                        key: GLOBAL_SEARCH_SUGGESTION_FIELDS_CONFIG[type],
                        search: term.term
                    }, true);
                } else {
                    return Observable.of([]);
                }
            }).catch(error => {
            console.log(error);
            this.loadingSuggestion = false;
            return Observable.of([]);
        }).subscribe(items => {
            this.loadingSuggestion = false;

            if (this.searchForm.value.inputControl) {
                let suggestions: Suggestion[] = items.map((searchIndex: SearchIndex) => {
                    searchIndex.value = searchIndex.value.substr(0, 100);
                    return {type: 'suggestion', term: this.searchForm.value.inputControl, details: searchIndex};
                }).slice(0, 5);

                suggestions = this.settings.hideSearchHistoryAndSuggestions ? [] : suggestions;
                let searchHistory = this.getSearchHistory();
                let searchInCategories = this.getSearchInCategoriesAsSuggestions();

                this.dropdownList = searchInCategories.concat(suggestions, searchHistory);
            }
        });

        this.searchForm.controls['inputControl'].valueChanges.subscribe(value => {
            this.searchTerms.next({term: value, category: this.settings.searchCategory});

            // if there is no input value and it's not a global search results screen - add navigation shortcuts
            if (!value && !this.settings.suppressBackRoute) {
                this.dropdownList = this.getDropdownNavigationShortcutsSearchInCategoriesAndSearchHistoriesAsSuggestions();
                this.resetSelectedItemIndex();
            } else {
                this.dropdownList = this.getSearchInCategoriesAndSearchHistoriesAsSuggestions();
                this.resetSelectedItemIndex();
            }
        });
    }

    public ngOnChanges(changes: SimpleChanges) {
        if (this.searchForm && changes.hasOwnProperty('settings') && this.hasChanged(changes['settings'].currentValue, changes['settings'].previousValue)) {
            // set values after reload page
            this.isSearchInputExpanded = this.settings.keepOpen;
            this.globalSearchService.toggleGlobalSearch.emit(this.isSearchInputExpanded);
            this.backRoute = this.settings.backRoute ? this.settings.backRoute : null;
            this.searchForm.controls['inputControl'].setValue(this.settings.searchString);

            if (!this.settings.keepOpen) {
                this.close();
            }
            if (changes['settings'].currentValue.searchCategory !== changes['settings'].previousValue.searchCategory) {
                this.setDropdownForEmptyInputValue(this.searchForm.controls['inputControl'].value, this.settings.searchCategory);
            }
        }
    }

    private hasChanged(first: Object, second: Object) {
        if (!!first !== !!second) {
            return true;
        }
        let firstKeys = Object.keys(first || {}).sort();
        let secondKeys = Object.keys(second || {}).sort();
        if (firstKeys.length !== secondKeys.length) {
            return true;
        }
        for (let i = 0; i < firstKeys.length; i++) {
            let firstKey = firstKeys[i];
            let secondKey = secondKeys[i];
            if (firstKey !== secondKey || first[firstKey] !== second[secondKey]) {
                return true;
            }
        }
        return false;
    }

    public ngOnDestroy() {
        this.globalSearchService.toggleGlobalSearch.emit(false);
    }

    // click outside dropdown and input
    @HostListener('document:click', ['$event'])
    public onClick(event) {
        if (this.isSearchInputExpanded && !this.elementRef.nativeElement.contains(event.target)) {
            this.setDropdownVisibility(false);
            this.resetSelectedItemIndex();

            if (!this.settings.keepOpen) {
                this.close();
            }
        }
    }

    public selectElement(index) {
        this.selectedItemIndex = index;
    }

    public isSelected(index) {
        return this.selectedItemIndex === index;
    }

    public onInputClick() {
        let inputControlValue = this.searchForm.controls['inputControl'].value;
        this.setDropdownVisibility(true);
        this.isSearchInputExpanded = true;
        this.globalSearchService.toggleGlobalSearch.emit(this.isSearchInputExpanded);
        this.searchTerms.next({term: inputControlValue, category: this.settings.searchCategory});

        if (inputControlValue || this.settings.suppressBackRoute) {
            this.dropdownList = this.getSearchInCategoriesAndSearchHistoriesAsSuggestions();
        } else {
            this.dropdownList = this.getDropdownNavigationShortcutsSearchInCategoriesAndSearchHistoriesAsSuggestions();
        }
    }

    public onCloseClick() {
        this.setDropdownVisibility(false);
        this.close();

        if (this.settings.applyLocalSearchQuery) {
            this.settings.applyLocalSearchQuery('', this.settings.searchCategory);
        }
    }

    public navigateTo(url: string) {
        this.router.navigate([url]);
        this.close();
    }

    public navigateBack() {
        this.router.navigateByUrl(this.backRoute);
    }

    public getTypeByCategory(category: string): string {
        switch (category) {
            case GLOBAL_SEARCH_TABS.JOBS.name:
                return SEARCH_INDEX_TYPE.SERVICES;
            case GLOBAL_SEARCH_TABS.APPLICATIONS.name:
                return SEARCH_INDEX_TYPE.APPLICATIONS;
            case GLOBAL_SEARCH_TABS.DEPLOYMENTS.name:
                return SEARCH_INDEX_TYPE.DEPLOYMENT;
            case GLOBAL_SEARCH_TABS.TEMPLATES.name:
                return SEARCH_INDEX_TYPE.TEMPLATES;
            case LOCAL_SEARCH_CATEGORIES.POLICIES.name:
                return  SEARCH_INDEX_TYPE.POLICIES;
            default:
                return '';
        }
    }

    public onSubmit(form?): void {
        this.settings.keepOpen = true;
        let term: string = (form && form.value.inputControl) ? form.value.inputControl : '';
        let category: string = this.settings.searchCategory;

        if (this.selectedItemIndex > (this.settings.suppressBackRoute ? this.searchInCategories.length - 1 : 0)) { // if user is in global search results, skip categories
            let item = this.dropdownList[this.selectedItemIndex];
            if (item.details.hasOwnProperty('name')) {
                category = item.details.name;
            }
            if (item.type === 'suggestion' || item.type === 'history') {
                term = this.dropdownList[this.selectedItemIndex].details.value || this.dropdownList[this.selectedItemIndex].details.key;
                this.searchForm.controls['inputControl'].setValue(term);
            }
            this.updateSearchHistoryListInLocalStorage(term, category);
            this.onSelect.emit(term);
        } else {
            this.updateSearchHistoryListInLocalStorage(form.value.inputControl, category);
        }

        // Hide the drop down
        this.setDropdownVisibility(false);
        if (!this.settings.suppressBackRoute) {
            this.backRoute = this.location.path();
        }

        if (this.settings.applyLocalSearchQuery &&
            !this.getSearchInCategoriesAsSuggestions().filter(item => JSON.stringify(item) === JSON.stringify(this.dropdownList[this.selectedItemIndex])).length) {
            this.settings.applyLocalSearchQuery(term, category);
            return;
        }

        this.resetSelectedItemIndex();
        term ? this.router.navigate(['/app/search', term, this.getRouteParams(category)]) :
            this.router.navigate(['/app/search', this.getRouteParams(category)]);
    }

    public hideNavigationAndSearchSuggestionItemsIfGlobalSearchResults(item: Suggestion) {
        return this.settings.applyLocalSearchQuery || !((item.type === 'navigationShortcut' || item.type === 'searchCategory') && this.settings.suppressBackRoute);
    }

    public checkIfItsNotGlobalSearchResultsAndSearchInputIsNotEmptyAndHistoryExist() {
        return !this.settings.suppressBackRoute && !this.searchForm.controls['inputControl'].value && this.dropdownList.filter(item => (item.type === 'history')).length;
    }

    public checkIfThereIsHistoryOrSuggestions() {
        let hasHistoryOrSuggestions = !!this.dropdownList.filter(item => (item.type === 'history' || item.type === 'suggestion')).length;
        return this.settings.suppressBackRoute ? !!hasHistoryOrSuggestions : this.isSearchInputExpanded;
    }

    // keyboard click event
    public keyEvent(event) {
        switch (event.keyCode) {
            case 13: // enter
                this.setDropdownVisibility(false);
                break;
            case 27: // ESC
                if (this.isSearchInputExpanded) {
                    this.setDropdownVisibility(false);
                    this.resetSelectedItemIndex();

                    if (!this.settings.keepOpen) {
                        this.close();
                        event.srcElement.blur();
                    }
                }
                break;
            case 38: // up arrow
                // added preventDefault() to prevent move cursor at the begin of string in input
                event.preventDefault();

                this.setDropdownVisibility(true);
                // skip dropdownNavigationShortcuts and searchInCategories
                if (this.selectedItemIndex <= this.itemsToSkip()) {
                    this.selectedItemIndex = this.dropdownList.length;
                }
                this.selectedItemIndex = this.selectedItemIndex > -1 ? this.selectedItemIndex - 1 : -1;
                break;
            case 40: // down arrow
                // added preventDefault() to prevent move cursor at the end of string in input
                event.preventDefault();

                this.setDropdownVisibility(true);
                // skip dropdownNavigationShortcuts and searchInCategories
                if (this.selectedItemIndex < this.itemsToSkip()) {
                    this.selectedItemIndex = this.itemsToSkip();
                }
                this.selectedItemIndex = this.selectedItemIndex < this.dropdownList.length - 1 ?
                    this.selectedItemIndex + 1 : (this.settings.suppressBackRoute ? this.searchInCategories.length : 0) - 1;
                break;
            default:
                this.setDropdownVisibility(true);
                break;
        }
    }

    private itemsToSkip() {
        let ifThereIsNoSearchInputSkipSearchInCategories = this.searchForm.value.inputControl ? 0 : this.searchInCategories.length;
        let ifGlobalSearchResultsSkipSearchInCategories = this.settings.suppressBackRoute ?
            this.searchInCategories.length : ifThereIsNoSearchInputSkipSearchInCategories;
        return (this.searchForm.value.inputControl || this.settings.suppressBackRoute ?
            0 : this.dropdownNavigationShortcuts.length) + ifGlobalSearchResultsSkipSearchInCategories - 1;
    }


    private initSearchHistoryList() {
        let currentStoreValue = localStorage.getItem(this.currentStoreKey);
        let store: Map<number, SearchHistory> = (currentStoreValue && !_.isEmpty(JSON.parse(currentStoreValue))) ?
            new Map<number, SearchHistory>(JSON.parse(currentStoreValue)) :
            new Map<number, SearchHistory>();
        this.searchHistoryList = store.get(this.currentStoreVersion) || new SearchHistory();
        Array.from(store.keys()).forEach(key => {
            if (key < this.currentStoreVersion) {
                store.delete(key);
            }
        });
    }

    private setDropdownVisibility(value) {
        this.isDropdownVisible = value;
    }

    private close() {
        this.isSearchInputExpanded = false;
        this.globalSearchService.toggleGlobalSearch.emit(this.isSearchInputExpanded);
        this.searchForm.controls['inputControl'].reset();
        this.dropdownList = [];
    }

    private updateSearchHistoryListInLocalStorage(value: string, category: string) {
        this.removeOldestOverTheLimit(category);
        if (value && _.findIndex(this.searchHistoryList[category], (item: SearchHistoryItem) => item.key === value) === -1) {
            this.searchHistoryList[category].push(new SearchHistoryItem(value, new Date().getTime()));
            this.searchHistoryList[category] = _.sortBy(this.searchHistoryList[category], 'key');

            let store: Map<number, SearchHistory> = new Map<number, SearchHistory>();
            store.set(this.currentStoreVersion, this.searchHistoryList);

            // save searchHistoryList for current path in local storage
            localStorage.setItem(this.currentStoreKey, JSON.stringify(Array.from(store.entries())));
        }
    }

    private removeOldestOverTheLimit(category: string) {
        if (this.searchHistoryList[category].length >= this.limit) {
            let minDateItem = _.minBy(this.searchHistoryList[category], 'addDate');
            let indexToRemove = this.searchHistoryList[category].indexOf(minDateItem);
            this.searchHistoryList[category].splice(indexToRemove, 1);
        }
    }

    private getRouteParams(searchCategory) {
        let params = {};
        if (searchCategory) {
            params['category'] = encodeURIComponent(searchCategory);
        }

        if (this.backRoute) {
            params['backRoute'] = encodeURIComponent(this.backRoute);
        }

        if (!this.filtersEmpty(this.settings.filters)) {
            params['filters'] = encodeURIComponent(JSON.stringify(this.settings.filters));
        }

        return ViewUtils.sanitizeRouteParams(params);
    }

    private filtersEmpty(filters: GlobalSearchFilters) {
        for (let filterType in filters) {
            if (filters.hasOwnProperty(filterType)) {
                for (let filter in filters[filterType]) {
                    if (filters[filterType].hasOwnProperty(filter) && filters[filterType][filter].length) {
                        return false;
                    }
                }
            }
        }
        return true;
    }

    private setDropdownForEmptyInputValue(value, category) {
        if (this.isSearchInputExpanded && (!value || !value.length)) {
            this.dropdownList = this.searchHistoryList[category].map(item => {
                return {type: 'history', details: item};
            });
        }
    }

    private getSearchHistory(): Suggestion[] {
        return this.settings.hideSearchHistoryAndSuggestions ? [] : this.getSearchHistoryAsSuggestions();
    }

    private getSearchHistoryAsSuggestions(): Suggestion[] {
        return this.searchHistoryList[this.settings.searchCategory].map(item => {
            return {type: 'history', details: item};
        });
    }

    private getSearchInCategoriesAsSuggestions(): Suggestion[] {
        return this.searchInCategories.map((category: any) => {
            return {type: 'searchCategory', term: this.searchForm.value.inputControl, details: category};
        });
    }

    private getDropdownNavigationShortcuts(): Suggestion[] {
        return this.dropdownNavigationShortcuts.map((category: any) => {
            return {type: 'navigationShortcut', term: this.searchForm.value.inputControl, details: category, icon: category.icon};
        });
    }

    private getDropdownNavigationShortcutsSearchInCategoriesAndSearchHistoriesAsSuggestions(): Suggestion[] {
        let dropdownNavigationShortcuts = this.getDropdownNavigationShortcuts();
        let searchInCategories = this.getSearchInCategoriesAsSuggestions();
        let searchHistory = this.getSearchHistory();

        return dropdownNavigationShortcuts.concat(searchInCategories, searchHistory);
    }

    private getSearchInCategoriesAndSearchHistoriesAsSuggestions(): Suggestion[] {
        let searchHistory = this.getSearchHistory();
        let searchInCategories = this.getSearchInCategoriesAsSuggestions();
        return searchInCategories.concat(searchHistory);
    }

    private resetSelectedItemIndex() {
        this.selectedItemIndex = -1;
    }
}
