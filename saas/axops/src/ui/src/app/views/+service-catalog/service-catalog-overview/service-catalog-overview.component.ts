import { Component, OnInit, OnDestroy } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { BehaviorSubject, Subscription } from 'rxjs';

import { TemplateService, ViewPreferencesService } from '../../../services';
import { Template, ViewPreferences } from '../../../model';
import { LayoutSettings } from '../../layout';
import { GLOBAL_SEARCH_TABS, GlobalSearchSetting, LaunchPanelService, BranchesFiltersComponent, ViewUtils } from '../../../common';

@Component({
    selector: 'ax-service-catalog',
    templateUrl: './service-catalog-overview.html',
})

export class ServiceCatalogOverviewComponent implements OnInit, OnDestroy, LayoutSettings {
    public loading: boolean = false;
    public onScrollLoading: boolean = false;
    public branchesFormattedSelection: string;
    public filters: string[] = [];

    public toolbarFilters = {
        data: [{
            name: 'Container',
            value: 'container',
            icon: { color: 'running' },
        }, {
            name: 'Workflow',
            value: 'workflow',
            icon: { color: 'success' },
        }, {
            name: 'Deployment',
            value: 'deployment',
            icon: { color: 'queued' },
        }],
        model: [],
        onChange: (data) => {
            this.router.navigate(['/app/service-catalog/overview', this.getRouteParams()]);
        }
    };
    public globalSearch: BehaviorSubject<GlobalSearchSetting> = new BehaviorSubject<GlobalSearchSetting>({
        suppressBackRoute: false,
        keepOpen: false,
        searchCategory: GLOBAL_SEARCH_TABS.TEMPLATES.name,
    });

    private selectedBranch: string = null;
    private selectedRepo: string = null;
    private canScroll: boolean = false;
    private serviceCatalog: Template[] = [];
    private getTemplatesSubscription: Subscription;
    private viewPreferences: ViewPreferences;
    private subscriptions: Subscription[] = [];

    constructor(
        private activatedRoute: ActivatedRoute,
        private templateService: TemplateService,
        private launchPanelService: LaunchPanelService,
        private router: Router,
        private viewPreferencesService: ViewPreferencesService) {
    }

    get pageTitle(): string {
        return 'Templates';
    }

    get breadcrumb(): { title: string, routerLink: any[] }[] {
        return ViewUtils.getBranchBreadcrumb(this.selectedRepo, this.selectedBranch, '/app/service-catalog/overview', this.viewPreferences);
    }

    public branchNavPanelUrl = '/app/service-catalog/overview';

    public async ngOnInit() {
        this.viewPreferences = await this.viewPreferencesService.getViewPreferences();
        this.activatedRoute.params.subscribe(params => {
            this.serviceCatalog = [];
            [this.selectedRepo, this.selectedBranch] = ViewUtils.getSelectedRepoBranch(params, this.viewPreferences);
            this.branchesFormattedSelection = BranchesFiltersComponent.formatSelection(this.selectedRepo, this.selectedBranch, 'templates');
            this.loading = true;
            if (params['filters']) {
                this.toolbarFilters.model = decodeURIComponent(params['filters']).split(',');
            }
            this.loading = true;
            this.loadTemplates(0);
            this.viewPreferencesService.updateViewPreferences(v => Object.assign(v.filterState, { selectedBranch: this.selectedBranch, selectedRepo: this.selectedRepo }));
        });

        this.subscriptions.push(this.viewPreferencesService.getViewPreferencesObservable().subscribe(viewPreferences => {
            if (viewPreferences.changeInfo && viewPreferences.changeInfo.viewFavoriteUpdated) {
                this.viewPreferences = viewPreferences;
                this.loading = true;
                this.serviceCatalog = [];
                this.loadTemplates(0);
            }
        }));
    }

    public ngOnDestroy() {
        this.getTemplatesUnsubscribe();
        this.subscriptions.forEach(item => item.unsubscribe());
        this.subscriptions = [];
    }

    public onScroll() {
        if (this.canScroll) {
            this.onScrollLoading = true;
            this.loadTemplates(this.serviceCatalog.length);
        }
    }

    public launchTemplate(template: Template) {
        this.templateService.getTemplateByIdAsync(template.id).subscribe(fullTemplate => {
            this.launchPanelService.openPanel(null, fullTemplate, false);
        });
    }

    private loadTemplates(offset: number) {
        this.getTemplatesUnsubscribe();
        this.canScroll = false;
        let pageSize = 20;
        this.getTemplatesSubscription = this.templateService.getTemplatesAsync({
            fields: ['id', 'name', 'description', 'repo', 'branch', 'type', 'cost'],
            repo: this.selectedRepo || null,
            branch: this.selectedBranch || null,
            limit: pageSize,
            offset: offset,
            type: this.toolbarFilters.model,
            repo_branch: this.viewPreferences.filterState.branches === 'my' ? this.viewPreferences.favouriteBranches.map(branch => {
                return { branch: branch.name, repo: branch.repo };
            }) : null,
        }, false).subscribe(success => {
            this.serviceCatalog = this.serviceCatalog.concat(success.data || []);
            this.canScroll = (success.data || []).length >= pageSize;
            this.loading = false;
            this.onScrollLoading = false;
        });
    }

    private getRouteParams(updatedParams?) {
        let params = {};
        if (this.selectedBranch) {
            params['branch'] = encodeURIComponent(this.selectedBranch);
        }
        if (this.selectedRepo) {
            params['repo'] = encodeURIComponent(this.selectedRepo);
        }
        if (this.toolbarFilters.model.length) {
            params['filters'] = encodeURIComponent(this.toolbarFilters.model.join(','));
        }

        return ViewUtils.sanitizeRouteParams(params, updatedParams);
    }

    private getTemplatesUnsubscribe() {
        if (this.getTemplatesSubscription) {
            this.getTemplatesSubscription.unsubscribe();
            this.getTemplatesSubscription = null;
        }
    }
}
