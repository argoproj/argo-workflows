import { Component, OnDestroy, OnInit } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { Location } from '@angular/common';
import { ReplaySubject, Subscription } from 'rxjs';

import { LayoutSettings } from '../../layout';
import { DropdownMenuSettings, NotificationsService } from 'argo-ui-lib/src/components';

import { Application, ACTIONS_BY_STATUS, ViewPreferences } from '../../../model';
import { ViewUtils, GLOBAL_SEARCH_TABS, GlobalSearchSetting } from '../../../common';
import { ApplicationsService, ModalService, ViewPreferencesService } from '../../../services';

@Component({
    selector: 'ax-app-overview',
    templateUrl: './app-overview.html',
    styles: [require('./app-overview.scss')],
})
export class AppOverviewComponent implements LayoutSettings, OnInit, OnDestroy {
    public searchString: string = null;
    public globalSearch: ReplaySubject<GlobalSearchSetting> = new ReplaySubject<GlobalSearchSetting>();
    private viewPreferences: ViewPreferences;

    public toolbarFilters = {
        data: [{
            name: 'Terminated',
            value: 'Terminated',
            icon: { color: 'running' },
        }, {
            name: 'Active',
            value: 'Active',
            icon: { color: 'success' },
        }, {
            name: 'Error',
            value: 'Error',
            icon: { color: 'fail' },
        }, {
            name: 'Stopped',
            value: 'Stopped',
            icon: { color: 'queued' },
        }],
        model: [],
        onChange: () => {
            this.router.navigate(['/app/applications', this.getRouteParams()]);
        }
    };

    private offset: number = 0;
    private bufferSize: number = 15;
    private dataLoaded: boolean = false;
    private canScroll: boolean = true;
    private firstLoading: boolean = true;
    private applications: Application[] = [];
    private getApplicationSubscription: Subscription;

    constructor(private route: ActivatedRoute,
                private router: Router,
                private location: Location,
                private modalService: ModalService,
                private notificationsService: NotificationsService,
                private applicationsService: ApplicationsService,
                private viewPreferencesService: ViewPreferencesService,
    ) {}

    public async ngOnInit() {
        this.viewPreferences = await this.viewPreferencesService.getViewPreferences();
        this.route.params.subscribe(params => {
            let viewPreferencesFilterState = this.viewPreferences.filterStateInPages['/app/applications'] || {};
            this.toolbarFilters.model = params['filters'] ? params['filters'].split(',') : viewPreferencesFilterState.filters || [];

            this.globalSearch.next({
                suppressBackRoute: false,
                keepOpen: false,
                searchCategory: GLOBAL_SEARCH_TABS.APPLICATIONS.name,
            });

            this.viewPreferencesService.updateViewPreferences(v => {
                v.filterStateInPages['/app/applications'] = {
                    filters: this.toolbarFilters.model,
                };
            });

            this.reset();
            this.getApplications();
        });
    }

    public ngOnDestroy() {
        this.unsubscribeGetApplication();
    }

    get pageTitle(): string {
        return 'Applications';
    }

    get hiddenToolbar(): boolean {
        return false;
    }

    get hasTabs(): boolean {
        return false;
    }

    get globalAddActionMenu(): DropdownMenuSettings {
        return new DropdownMenuSettings([{
            title: 'Bulk Actions',
            iconName: '',
            action: async () => {
                this.navigateToBulkAction();
            },
        }], 'fa-ellipsis-v');
    }

    get breadcrumb(): { title: string, routerLink?: any[] }[] {
        return [{
            title: 'All Applications',
            routerLink: null,
        }];
    };

    get layoutSettings(): LayoutSettings {
        return this;
    }

    public changeView(view: string) {
        this.router.navigate(['/app/applications', this.getRouteParams({view})]);
    }

    public onScroll() {
        if (this.canScroll) {
            this.canScroll = false;
            this.dataLoaded = false;
            this.getApplications(true);
        }
    }

    public get applicationMenuCreator() {
        return (application: Application) => {
            let items: {title: string, iconName: string, action: () => any}[] = [];
            if (ACTIONS_BY_STATUS.START.indexOf(application.status) > -1) {
                items.push({
                    title: 'Start', iconName: '',
                    action: () => this.runAction(
                        'Start Application',
                        'Are you sure you want to start application?',
                        'Application has been successfully started',
                        () => this.applicationsService.startApplication(application.id).toPromise()),
                });
            }
            if (ACTIONS_BY_STATUS.STOP.indexOf(application.status) > -1) {
                items.push({
                    title: 'Stop', iconName: '',
                    action: () => this.runAction(
                        'Stop Application',
                        'Are you sure you want to stop application?',
                        'Application has been successfully stopped',
                        () => this.applicationsService.stopApplication(application.id).toPromise()),
                });
            }
            if (ACTIONS_BY_STATUS.TERMINATE.indexOf(application.status) > -1) {
                items.push({
                    title: 'Terminate', iconName: '',
                    action: () => this.runAction(
                        'Terminate Application',
                        'Are you sure you want to terminate application?',
                        'Application has been successfully terminated',
                        () => this.applicationsService.deleteAppById(application.id).toPromise()),
                });
            }
            return new DropdownMenuSettings(items, 'fa-ellipsis-v');
        };
    }

    private runAction(title: string, confirmation: string, success: string, action: () => Promise<any>) {
        this.modalService.showModal(title, confirmation).subscribe(async confirmed => {
            if (confirmed) {
                await action();
                this.notificationsService.success(success);
            }
        });
    }

    private navigateToBulkAction() {
        this.router.navigate(['/app/search', { category: 'applications', backRoute: this.location.path() }]);
    }

    private getRouteParams(updatedParams?) {
        let params = {
            filters: this.toolbarFilters.model.length > 0 ? this.toolbarFilters.model.join(',') : null,
        };

        if (this.searchString) {
            params['search'] = encodeURIComponent(this.searchString);
        }

        return ViewUtils.sanitizeRouteParams(params, updatedParams);
    }

    private getApplications(hideLoader: boolean = true) {
        this.unsubscribeGetApplication();

        this.getApplicationSubscription = this.applicationsService.getApplications({
            limit: this.bufferSize,
            offset: this.offset,
            include_details: false,
            search: this.searchString || '',
            status: this.toolbarFilters.model.join(',') || '',
            sort: 'status',
        }, hideLoader).subscribe(success => {
            this.firstLoading = false;
            this.applications = this.applications.concat(success);
            this.dataLoaded = true;
            this.canScroll = success.length === this.bufferSize;
            this.offset = this.offset + success.length;
        }, () => {
            this.dataLoaded = true;
        });
    }

    private unsubscribeGetApplication() {
        if (this.getApplicationSubscription) {
            this.getApplicationSubscription.unsubscribe();
            this.getApplicationSubscription = null;
        }
    }

    private reset() {
        this.offset = 0;
        this.bufferSize = 15;
        this.dataLoaded = false;
        this.canScroll = true;
        this.firstLoading = true;
        this.applications = [];
    }
}
