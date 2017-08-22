import { Component, Input, OnChanges, OnDestroy } from '@angular/core';
import { Router } from '@angular/router';
import { Subscription } from 'rxjs';

import { NotificationsService } from 'argo-ui-lib/src/components';
import { ApplicationFieldNames, Application, ACTIONS_BY_STATUS } from '../../../model';
import { ApplicationsService, GlobalSearchService, ModalService } from '../../../services';
import { Pagination, ApplicationsFilters } from '../../../common';
import { BulkUpdater } from '../bulk-updater';

@Component({
    selector: 'ax-applications-list',
    templateUrl: './applications-list.html',
})
export class ApplicationsListComponent implements OnChanges, OnDestroy {

    @Input()
    public filters: ApplicationsFilters;

    @Input()
    public searchString: string;

    public bulkUpdater: BulkUpdater<Application>;
    public limit: number = 10;
    public items: Application[] = [];
    public params: ApplicationsFilters;
    public dataLoaded: boolean = false;
    public pagination: Pagination = {
        limit: this.limit,
        offset: 0,
        listLength: this.items.length
    };

    private subscriptions: Subscription[] = [];

    constructor(private router: Router,
                private applicationsService: ApplicationsService,
                private globalSearchService: GlobalSearchService,
                modalService: ModalService,
                notificationsService: NotificationsService) {
        this.bulkUpdater = new BulkUpdater<Application>(modalService, notificationsService).
            addAction('terminate', {
                title: 'Terminate Applications',
                confirmation: count => this.actionConfirmation(count, 'terminate'),
                execute: deployment => this.applicationsService.deleteAppById(deployment.id, false).toPromise(),
                isApplicable: deployment => this.isApplicationInStatus(deployment, ACTIONS_BY_STATUS.TERMINATE),
                warningMessage: deployments => this.formatWarningMessage(deployments, ACTIONS_BY_STATUS.TERMINATE),
                postMessage: (successfulCount, failedCount) => this.postActionMessage(successfulCount, failedCount, 'terminated')
            }).addAction('stop', {
                title: 'Stop Applications',
                confirmation: count => this.actionConfirmation(count, 'stop'),
                execute: deployment => this.applicationsService.stopApplication(deployment.id, false).toPromise(),
                isApplicable: deployment => this.isApplicationInStatus(deployment, ACTIONS_BY_STATUS.STOP),
                warningMessage: deployments => this.formatWarningMessage(deployments, ACTIONS_BY_STATUS.STOP),
                postMessage: (successfulCount, failedCount) => this.postActionMessage(successfulCount, failedCount, 'stopped')
            }).addAction('start', {
                title: 'Start Applications',
                confirmation: count => this.actionConfirmation(count, 'start'),
                execute: deployment => this.applicationsService.startApplication(deployment.id, false).toPromise(),
                isApplicable: deployment => this.isApplicationInStatus(deployment, ACTIONS_BY_STATUS.START),
                warningMessage: deployments => this.formatWarningMessage(deployments, ACTIONS_BY_STATUS.START),
                postMessage: (successfulCount, failedCount) => this.postActionMessage(successfulCount, failedCount, 'started')
            });
        this.bulkUpdater.actionExecuted.subscribe(action => this.updateItems(this.params, this.pagination, true));
    }

    private actionConfirmation(count: number, verb: string) {
        return `Are you sure you want to ${verb} ${count} Application${count > 1 ? 's' : ''}?`;
    }

    private postActionMessage(successfullCount: number, failedCount: number, verb: string) {
        let beVerb = successfullCount === 1 ? 'was' : 'were';
        let entity = successfullCount === 1 ? 'application' : 'applications';
        return `${successfullCount} ${entity} ${beVerb} successfully ${verb}`;
    }

    private isApplicationInStatus(deployment: Application, statuses: string[]) {
        return statuses.indexOf(deployment.status) > -1;
    }

    private formatWarningMessage(apps: Application[], statuses: string[]) {
        return `The action is applicable only for applications in status ${ statuses.join(', ') } and will not be performed on ` +
               `${ apps.length } application${ apps.length > 1 ? 's' : ''}`;
    }

    public ngOnChanges() {
        this.params = {
            application_statuses: this.filters.application_statuses
        };

        // restart pagination if changed search parameters
        this.pagination = {limit: this.limit, offset: 0, listLength: this.items.length};
        this.updateItems(this.params, this.pagination, true);
    }

    public ngOnDestroy() {
        this.subscriptions.forEach(subscription => subscription.unsubscribe());
        this.subscriptions = [];
    }

    public onPaginationChange(pagination: Pagination) {
        this.limit = pagination.limit;
        this.updateItems(this.params, { offset: pagination.offset, limit: pagination.limit }, true);
        this.bulkUpdater.clearSelection();
    }

    public navigateToDetails(item): void {
        this.globalSearchService.navigate(['/app/applications/details/' + item.id]);
    }

    private updateItems(params: any, pagination: Pagination, hideLoader?: boolean) {
        this.dataLoaded = false;
        pagination.limit += 1;

        this.subscriptions.push(this.getApplications(params, pagination, hideLoader).subscribe(result => {
            this.dataLoaded = true;

            this.items = result.slice(0, this.limit) || [];
            this.bulkUpdater.items = this.items;

            this.pagination = {
                offset: pagination.offset,
                limit: this.limit,
                listLength: this.items.length,
                hasMore: result.length > this.limit
            };
        }, error => {
            this.dataLoaded = true;
            this.items = [];
        }));
    }

    private getApplications(params: any, pagination: Pagination, hideLoader?: boolean) {
        let parameters = {
            limit: null,
            offset: null,
            include_details: false,
            search: this.searchString || '',
            status: null || '',
            searchFields: [
                ApplicationFieldNames.name,
                ApplicationFieldNames.endpoints,
                ApplicationFieldNames.status
            ],
            sort: 'status',
        };

        if (pagination.offset) {
            parameters.offset = pagination.offset;
        }

        if (pagination.limit) {
            parameters.limit = pagination.limit;
        }

        if (params.application_statuses && params.application_statuses.length) {
            parameters.status = params.application_statuses;
        }

        return this.applicationsService.getApplications(parameters, hideLoader);
    }
}
