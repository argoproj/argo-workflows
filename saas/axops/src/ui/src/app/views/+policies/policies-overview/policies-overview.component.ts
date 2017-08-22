import { Component, OnInit, OnDestroy } from '@angular/core';
import { Router, ActivatedRoute } from '@angular/router';
import { Subscription, BehaviorSubject } from 'rxjs';

import { PoliciesService, ModalService, ViewPreferencesService} from '../../../services';
import { Policy, ViewPreferences } from '../../../model';
import { LayoutSettings } from '../../layout';
import { ViewUtils, GlobalSearchSetting, LOCAL_SEARCH_CATEGORIES, BranchesFiltersComponent } from '../../../common';

@Component({
    selector: 'ax-policies-overview',
    templateUrl: './policies-overview.html',
    styles: [ require('./policy-overview.scss') ],
})
export class PoliciesOverviewComponent implements OnInit, OnDestroy, LayoutSettings {
    public searchString: string = '';
    public allSelected: boolean = false;
    public selectedItems: number = 0;
    public loading: boolean = false;
    public onScrollLoading: boolean = false;
    public branchesFormattedSelection: string;
    public invalidPolicies: Policy[] = [];
    public showInvalidPanel: boolean = false;
    public invalidOnly: boolean = false;
    public invalidDetailsCollapsed: boolean = false;
    public branchNavPanelUrl = '/app/policies/overview';
    public toolbarFilters = this.resetToolbarFilters();

    private selectedRepo: string = null;
    private selectedBranch: string = null;
    private policies: Policy[] = [];
    private canScroll: boolean = false;
    private getPoliciesSubscription: Subscription;
    private subscriptions: Subscription[] = [];
    private viewPreferences: ViewPreferences;

    constructor(private router: Router,
                private activatedRoute: ActivatedRoute,
                private policiesService: PoliciesService,
                private modalService: ModalService,
                private viewPreferencesService: ViewPreferencesService) {
    }

    public globalSearch: BehaviorSubject<GlobalSearchSetting> = new BehaviorSubject<GlobalSearchSetting>({
        suppressBackRoute: false,
        keepOpen: false,
        searchCategory: LOCAL_SEARCH_CATEGORIES.POLICIES.name,
        searchString: this.searchString,
        applyLocalSearchQuery: (searchString) => {
            this.clearSelectedPolicies();
            this.searchString = searchString;
            this.router.navigate(['app/policies/overview', this.getRouteParams()]);
        },
    });

    public async ngOnInit() {
        this.viewPreferences = await this.viewPreferencesService.getViewPreferences();
        this.activatedRoute.params.subscribe(async params => {
            this.policies = [];
            this.showInvalidPanel = params['invalid'] ? false : true;
            this.invalidOnly = params['invalid'] ? true : false;

            this.selectedRepo = params['repo'] ? decodeURIComponent(params['repo']) : null;
            this.selectedBranch = params['branch'] ? decodeURIComponent(params['branch']) : null;
            this.searchString = params['search'] ? decodeURIComponent(params['search']) : null;
            this.branchesFormattedSelection = BranchesFiltersComponent.formatSelection(this.selectedRepo, this.selectedBranch, 'policies');

            if (this.invalidOnly) {
                this.toolbarFilters = null;
            } else {
                if (!this.toolbarFilters) {
                    this.toolbarFilters = this.resetToolbarFilters();
                }
                this.toolbarFilters.model = params['filters'] ?
                    decodeURIComponent(params['filters']).split(',') : [];
                [this.selectedRepo, this.selectedBranch] = ViewUtils.getSelectedRepoBranch(params, this.viewPreferences);
            }


            if (this.searchString) {
                this.globalSearch.value.searchString = this.searchString;
                this.globalSearch.value.keepOpen = true;
            }

            this.loading = true;
            this.getPolicies(0);
            this.viewPreferencesService.updateViewPreferences(v => Object.assign(v.filterState, { selectedBranch: this.selectedBranch, selectedRepo: this.selectedRepo }));

            if (!this.invalidOnly || (this.toolbarFilters && !(this.toolbarFilters.model.length === 1 &&
                this.toolbarFilters.model.indexOf('enabled') > -1))) {
                this.getInvalidPolicies();
            } else if (this.invalidOnly) {
                this.invalidPolicies = this.policies;
            }
        });

        this.subscriptions.push(this.viewPreferencesService.getViewPreferencesObservable().subscribe(viewPreferences => {
            if (viewPreferences.changeInfo && viewPreferences.changeInfo.viewFavoriteUpdated) {
                this.viewPreferences = viewPreferences;
                this.loading = true;
                this.policies = [];
                this.getPolicies(0);
            }
        }));
    }

    public ngOnDestroy() {
        this.getPoliciesUnsubscribe();
        this.subscriptions.forEach(item => item.unsubscribe());
        this.subscriptions = [];
    }

    get pageTitle(): string {
        return 'Policies';
    }

    get breadcrumb(): { title: string, routerLink: any[] }[] {
        let arr = ViewUtils.getBranchBreadcrumb(
            this.selectedRepo, this.selectedBranch, '/app/policies/overview', this.viewPreferences,
            ( this.invalidOnly ? 'Invalid Policies' : null), this.invalidOnly);

        return arr;
    }

    public resetToolbarFilters() {
        return {
            data: [{
                value: 'enabled',
                name: 'Enabled',
                icon: {},
            }, {
                value: 'disabled',
                name: 'Disabled',
                icon: { color: 'queued' },
            }],
            model: [],
            onChange: (data) => {
                this.router.navigate(['app/policies/overview', this.getRouteParams()]);
                this.showInvalidPanel = false;
            }
        };
    }

    public clearSelectedPolicies() {
        this.policies.filter(item => item.selected).forEach(item => item.selected = false);
        this.allSelected = false;
        this.selectedItems = 0;
    }

    public selectPolicy(policy) {
        this.allSelected = false;
        policy.selected = !policy.selected;

        policy.selected ? this.selectedItems++ : this.selectedItems--;
        this.showInvalidPanel = false;
    }

    public selectAllPolicies() {
        this.allSelected = !this.allSelected;
        this.selectedItems = 0;
        this.showInvalidPanel = false;

        this.policies.forEach(p => {
            p.selected = this.allSelected;

            if (this.allSelected) {
                this.selectedItems++;
            }
        });
    }

    public enableSelectedPolicies(isEnabled: boolean) {
        this.modalService.showModal(`${isEnabled ? 'Enabled' : 'Disable'} policy`,
            `Are you sure you want to ${isEnabled ? 'enable' : 'disable'} ${this.selectedItems} policies?`)
            .subscribe(result => {
                if (result) {
                    this.allSelected = false;
                    this.selectedItems = 0;
                    this.policies.filter(p => p.selected).forEach(p => {
                        this.setPolicyStatus(p, isEnabled);
                        p.selected = false;
                    });
                }
            });
    }

    public deleteSelectedInvalidPolicies(isEnabled: boolean) {
        let selectedInvalidPolices = this.policies.filter(p => p.selected && p.status === 'invalid');
        if (selectedInvalidPolices.length) {
            this.modalService.showModal(`Delete policy`,
                `Only invalid policy can be deleted. \nAre you sure you want to delete ${selectedInvalidPolices.length} policies?`)
                .subscribe(result => {
                    if (result) {
                        this.allSelected = false;
                        this.selectedItems = 0;
                        let promiseArr = [];
                        selectedInvalidPolices.forEach(p => {
                            promiseArr.push(this.policiesService.deletePolicy(p.id).toPromise());
                        });
                        Promise.all(promiseArr).then(res => {
                            this.router.navigate(['app/policies/overview', this.getRouteParams()]);
                        }, err => {
                            this.router.navigate(['app/policies/overview', this.getRouteParams()]);
                        });
                    }
                });
        } else {
            this.modalService.showModal(`Cannot delete policy`,
                `You can only delete invalid policy.`, '', { name: null, color: null }, true);
        }
    }

    public getPolicies(offset: number) {
        this.getPoliciesUnsubscribe();
        let pageSize = 20;
        let params = this.composeGetParams(offset, pageSize, this.invalidOnly);

        this.canScroll = false;
        this.getPoliciesSubscription = this.policiesService.getPolicies(params, true).subscribe(results => {
            this.policies = this.policies.concat(results.data || []);
            this.canScroll = (results.data || []).length >= pageSize;
            this.allSelected = (results.data || []).length === 0;
            this.loading = false;
            this.onScrollLoading = false;
            if (this.invalidOnly) {
                this.invalidPolicies = this.policies;
            }
        });
    }

    public getInvalidPolicies() {
        let params = this.composeGetParams(0, 0, true);

        this.policiesService.getPolicies(params, true).subscribe(results => {
            this.invalidPolicies = results.data || [];
        });
    }

    public composeGetParams(offset: number = 0, pageSize = 0, invalid?) {
        let showMyOnly = this.viewPreferences.filterState.branches === 'my';
        let enabled = undefined;
        let status = invalid ? 'invalid' : null;
        let enabledSelected = this.toolbarFilters && this.toolbarFilters.model.indexOf('enabled') > -1 ? true : undefined;
        let disabledSelected = this.toolbarFilters && this.toolbarFilters.model.indexOf('disabled') > -1 ? true : undefined;
        if ((enabledSelected && disabledSelected) || (!enabledSelected && !disabledSelected)) {
            enabled = undefined;
        } else if (enabledSelected && !disabledSelected) {
            enabled = true;
        } else if (!enabledSelected && disabledSelected) {
            enabled = false;
        }

        return {
            search: this.searchString,
            enabled: enabled,
            limit: pageSize,
            offset: offset,
            repo: this.selectedRepo,
            branch: this.selectedBranch,
            repo_branch: showMyOnly ? this.viewPreferences.favouriteBranches.map(branch => {
                return { branch: branch.name, repo: branch.repo };
            }) : null,
            status: status,
        };
    }

    public goToDetails(policy: Policy): void {
        this.router.navigate([`/app/policies/details/${policy.id}`]);
    }

    public onScroll() {
        if (this.canScroll) {
            this.onScrollLoading = true;
            this.getPolicies(this.policies.length);
        }
    }

    private setPolicyStatus(policy: Policy, isEnabled: boolean) {
        if (isEnabled) {
            this.policiesService.enablePolicy(policy.id).subscribe(success => policy.enabled = true);
        } else {
            this.policiesService.disablePolicy(policy.id).subscribe(success => policy.enabled = false);
        }
    }

    private getRouteParams(updatedParams?) {
        let params = {};
        if (this.selectedBranch) {
            params['branch'] = encodeURIComponent(this.selectedBranch);
        }
        if (this.selectedRepo) {
            params['repo'] = encodeURIComponent(this.selectedRepo);
        }
        if (this.searchString) {
            params['search'] = encodeURIComponent(this.searchString);
        }
        if (this.toolbarFilters && this.toolbarFilters.model.length > 0) {
            params['filters'] = encodeURIComponent(this.toolbarFilters.model.join(','));
        }

        return ViewUtils.sanitizeRouteParams(params, updatedParams);
    }

    private getPoliciesUnsubscribe() {
        if (this.getPoliciesSubscription) {
            this.getPoliciesSubscription.unsubscribe();
            this.getPoliciesSubscription = null;
        }
    }
}
