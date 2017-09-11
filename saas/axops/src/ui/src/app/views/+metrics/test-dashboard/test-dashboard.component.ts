import * as _ from 'lodash';
import { Observable } from 'rxjs';
import { Component, OnInit, ViewChild } from '@angular/core';
import { ActivatedRoute, Params, Router } from '@angular/router';

import { LayoutSettings, HasLayoutSettings } from '../../layout';
import { TaskStatus, LABEL_TYPES, CustomView, CustomViewInfo, CUSTOM_VIEW_TYPES } from '../../../model';
import { TaskService, CustomViewService, ModalService } from '../../../services';
import { SortOperations } from '../../../common/sortOperations/sortOperations';

import { BranchesFiltersComponent, LabelsFiltersComponent, TemplatesFiltersComponent } from '../../../common';
import { NotificationsService, DateRange } from 'argo-ui-lib/src/components';

declare let d3: any;

interface TemplateInfo {
    name: string;
    repo: string;
    cost: number;
    statusToCount: Map<TaskStatus, number>;
    chartData: any[];
}

const SORT_BY = {
    spending: 'spending',
    name: 'name'
};


@Component({
    selector: 'ax-test-dashboard',
    templateUrl: './test-dashboard.html',
    styles: [ require('./test-dashboard.scss') ],
})
export class TestDashboardComponent implements HasLayoutSettings, OnInit {
    chartOptions: any;
    range: DateRange;
    templatesInfo: TemplateInfo[] = [];
    customViews: CustomView[] = [];
    filtersPanel: boolean = false;
    selectedRepo: string = null;
    selectedBranch: string = null;
    selectedLabels: string = null;
    selectedTemplateNames: string = null;
    selectedCustomView: CustomView = null;
    selectedCustomViewName: string = null;
    filterInfo: string = '';
    labelType: string = LABEL_TYPES.service;
    showCustomViewPopup: boolean = false;
    sortBy: string = '';
    isInitialized = false;
    updateRequired = false;

    @ViewChild(BranchesFiltersComponent)
    branchesFilter: BranchesFiltersComponent;

    @ViewChild(LabelsFiltersComponent)
    labelsFilter: LabelsFiltersComponent;

    @ViewChild(TemplatesFiltersComponent)
    templatesFilter: TemplatesFiltersComponent;

    constructor(private router: Router,
                private activatedRoute: ActivatedRoute,
                private taskService: TaskService,
                private customViewService: CustomViewService,
                private modalService: ModalService,
                private notificationsService: NotificationsService) {

        activatedRoute.params.subscribe(params => {
            this.configureFilters(params);

            if (!this.isInitialized || this.updateRequired) {
                this.isInitialized = true;
                this.loadTemplateInfo(this.range).subscribe(templatesInfo => {
                    this.templatesInfo = SortOperations.sortBy(templatesInfo,
                        this.sortBy === SORT_BY.spending ? 'cost' : this.sortBy);
                    // sort by cost descending
                    if (this.sortBy === SORT_BY.spending) {
                        this.templatesInfo = this.templatesInfo.reverse();
                    }
                });
                this.loadCustomViews();
                this.selectedCustomView = new CustomView();
            }
        });
    }

    onDateRangeChange(range: DateRange) {
        this.navigate(range.toRouteParams());
    }

    get layoutSettings(): LayoutSettings {
        return {
            pageTitle: 'Metrics',
            hasExtendedBg: true,
            breadcrumb: [{
                title: `Metrics for ${this.range.format()}`,
                routerLink: null
            }]
        };
    }

    ngOnInit() {
        this.chartOptions = {
            chart: {
                type: 'pieChart',
                height: 120,
                margin: { top: 0, left: 0, right: 0, bottom: 0 },
                showLabels: false,
                duration: 500,
                labelThreshold: 0.01,
                labelSunbeamLayout: true,
                showLegend: false,
                donut: true,
                donutRatio: 0.54,
                tooltip: { enabled: false },
                color: ['#18BE94', '#E96D76', '#FBB465', '#0DADEA', '#ccc']
            }
        };
    }

    getCount(status: TaskStatus, info: TemplateInfo) {
        return info.statusToCount.get(status) || 0;
    }

    getPercent(status: TaskStatus, info: TemplateInfo) {
        return (this.getCount(status, info) / _.sum(Array.from(info.statusToCount.values())) * 100).toFixed();
    }

    get taskStatus(): any {
        return TaskStatus;
    }

    get isFiltered(): boolean {
        return (this.selectedBranch != null || this.selectedRepo != null ||
            this.selectedLabels != null || this.selectedTemplateNames != null);
    }

    byName(info: TemplateInfo) {
        return info.name;
    }

    byCustomViewName(customView: CustomView) {
        return customView.name;
    }

    showFiltersPanel() {
        this.navigate({ filtersPanel: true });
    }

    applyFilters() {
        let params = <any>{ filtersPanel: false };
        params.repo = this.branchesFilter.selectedRepo ? encodeURIComponent(this.branchesFilter.selectedRepo) : null;
        params.branch = this.branchesFilter.selectedBranch ? encodeURIComponent(this.branchesFilter.selectedBranch) : null;
        params.labels = this.labelsFilter.selectedLabels ? encodeURIComponent(this.labelsFilter.selectedLabels) : null;
        params.template_name = this.templatesFilter.selectedTemplateNames ? encodeURIComponent(this.templatesFilter.selectedTemplateNames) : null;
        this.navigate(params);
    }

    applyCustomView(customView: CustomView) {
        this.selectedCustomViewName = customView.name;
        let params = <any>{ filtersPanel: false };
        let customViewInfo: CustomViewInfo = JSON.parse(customView.info);
        params.repo = customViewInfo.repo ? encodeURIComponent(customViewInfo.repo) : null;
        params.branch = customViewInfo.branch ? encodeURIComponent(customViewInfo.branch) : null;
        params.labels = customViewInfo.labels ? encodeURIComponent(customViewInfo.labels) : null;
        params.template_name = customViewInfo.template_name ? encodeURIComponent(customViewInfo.template_name) : null;
        params.custom_view = customView.name ? encodeURIComponent(customView.name) : null;
        this.navigate(params);
    }

    applySortBySpending() {
        let params = <any>{ filtersPanel: false };
        params.sort_by = SORT_BY.spending;
        this.navigate(params);
    }

    applySortByName() {
        let params = <any>{ filtersPanel: false };
        params.sort_by = SORT_BY.name;
        this.navigate(params);
    }

    closePanel() {
        this.navigate({ filtersPanel: false });
    }

    onFilterTabChange(event: any) {
        if (event.selectedTab.tabKey === 'service-templates') {
            if (!this.templatesFilter.templates.length) {
                this.templatesFilter.loadTemplates();
            }
        }
    }

    saveCustomView() {
        this.selectedCustomView = new CustomView();
        if (this.selectedCustomViewName) {
            this.selectedCustomView = this.customViews.find(c => {
                return c.name === this.selectedCustomViewName;
            });

        }
        this.selectedCustomView.type = CUSTOM_VIEW_TYPES.testDashboard;
        let info = new CustomViewInfo();
        info.repo = this.selectedRepo;
        info.branch = this.selectedBranch;
        info.labels = this.selectedLabels;
        info.template_name = this.selectedTemplateNames;
        this.selectedCustomView.info = JSON.stringify(info);
        this.showCustomViewPopup = true;
    }

    clearFilters() {
        let params = <any>{ filtersPanel: false };
        params.repo = null;
        params.branch = null;
        params.labels = null;
        params.template_name = null;
        params.custom_view = null;
        this.navigate(params);
    }

    closeCustomViewPopup(customView?: CustomView) {
        if (customView) {
            // Reload custom views
            this.loadCustomViews();
            this.selectedCustomViewName = customView.name;
            this.navigate({custom_view: this.selectedCustomViewName});
        }

        this.showCustomViewPopup = false;
    }

    deleteCustomView(event: any, customView: CustomView): void {
        event.stopPropagation();
        this.modalService.showModal('Delete custom view',
            `Are you sure you want to delete custom view ${customView.name}?`)
            .subscribe(result => {
                if (result) {
                    if (this.selectedCustomViewName === customView.name) {
                        this.navigate({custom_view: null});
                    }
                    this.customViewService.deleteCustomView(customView.id, true).subscribe(() => {
                        this.loadCustomViews(true);
                        this.notificationsService.success(`Custom view ${customView.name} has been removed.`);
                    }, () => {
                        this.notificationsService.error(`Unable to remove custom view ${customView.name}.`);
                    });
                }
            });
    }

    private navigate(params: any) {
        let routeParams = <any>{};
        _.extend(routeParams, this.activatedRoute.snapshot.params, params);
        for (let name in routeParams) {
            if (routeParams.hasOwnProperty(name) && routeParams[name] === null) {
                delete routeParams[name];
            }
        }
        this.router.navigate(['app/metrics', routeParams]);
    }

    private loadTemplateInfo(range: DateRange): Observable<TemplateInfo[]> {
        let labelsQ = this.selectedLabels;
        if (this.selectedLabels && this.selectedLabels.indexOf(';') > -1) {
            labelsQ = this.selectedLabels.split(';').join('%3B');
        }
        return this.taskService.getTasks({
            startTime: this.range.startDate,
            endTime: this.range.endDate,
            branch: this.selectedBranch,
            repo: this.selectedRepo,
            // Note: backend api only expects ';' to be encoded not the ':'
            labels: labelsQ,
            template_name: this.selectedTemplateNames,
            fields: ['commit', 'template', 'status']
        }, false).map(res => {
            let templateToInfo = new Map<string, TemplateInfo>();
            res.data.forEach(job => {
                if (!job.hasOwnProperty('template')) {
                    return;
                }

                let key = `${job.template.repo}_${job.template.name}`;
                let info = templateToInfo.get(key) || {
                    name: job.template.name,
                    repo: job.template.repo,
                    cost: 0,
                    statusToCount: new Map<TaskStatus, number>(),
                    chartData: []
                };
                templateToInfo.set(key, info);
                info.statusToCount.set(job.status, (info.statusToCount.get(job.status) || 0) + 1);
                info.cost += ( job.template.cost || 0 );
            });
            let result = Array.from(templateToInfo.values());
            result.forEach(info => {
                info.chartData = [TaskStatus.Success, TaskStatus.Failed, TaskStatus.Init, TaskStatus.Running, TaskStatus.Waiting].map(
                    status => ({
                        y: this.getCount(status, info)
                    }));
            });
            this.filterInfo = this.selectedBranch;
            return result;
        });
    }

    private loadCustomViews(hideLoader?: boolean) {
        return this.customViewService.getCustomViews({ type: CUSTOM_VIEW_TYPES.testDashboard }, hideLoader).subscribe(result => {
            this.customViews = result;
        });
    }

    private configureFilters(params: Params): void {
        this.updateRequired = false;
        this.filtersPanel = params['filtersPanel'] === 'true';
        let range = DateRange.fromRouteParams(params);

        let selectedBranch = params['branch'] ? decodeURIComponent(params['branch']) : null;
        let selectedRepo = params['repo'] ? decodeURIComponent(params['repo']) : null;
        let selectedLabels = params['labels'] ? decodeURIComponent(params['labels']) : null;
        let selectedTemplateNames = params['template_name'] ? decodeURIComponent(params['template_name']) : null;
        let selectedCustomViewName = params['custom_view'] ? decodeURIComponent(params['custom_view']) : null;
        let sortBy = params['sort_by'] ? decodeURIComponent(params['sort_by']) : SORT_BY.spending;

        if (!DateRange.equals(range, this.range) ||
            selectedBranch !== this.selectedBranch ||
            selectedRepo !== this.selectedRepo ||
            selectedLabels !== this.selectedLabels ||
            selectedTemplateNames !== this.selectedTemplateNames ||
            selectedCustomViewName !== this.selectedCustomViewName ||
            sortBy !== this.sortBy
        ) {
            this.updateRequired = true;
        }

        this.selectedBranch = selectedBranch;
        this.selectedRepo = selectedRepo;
        this.selectedLabels = selectedLabels;
        this.selectedTemplateNames = selectedTemplateNames;
        this.range = range;
        this.selectedCustomViewName = selectedCustomViewName;
        this.sortBy = sortBy;
    }
}
