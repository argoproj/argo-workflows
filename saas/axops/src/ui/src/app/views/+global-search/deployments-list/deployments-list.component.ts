import { Component, Input, OnChanges, OnDestroy } from '@angular/core';
import { Router } from '@angular/router';
import { Subscription } from 'rxjs';

import { NotificationsService } from 'argo-ui-lib/src/components';
import { Deployment, POD_PHASE, ApplicationFieldNames, ACTIONS_BY_STATUS } from '../../../model';
import { DeploymentsService, GlobalSearchService, ModalService } from '../../../services';
import { Pagination, DeploymentsFilters } from '../../../common';
import { BulkUpdater } from '../bulk-updater';

@Component({
    selector: 'ax-deployments-list',
    templateUrl: './deployments-list.html',
})
export class DeploymentsListComponent implements OnChanges, OnDestroy {

    @Input()
    public filters: DeploymentsFilters;

    @Input()
    public searchString: string;

    public bulkUpdater: BulkUpdater<Deployment>;
    public limit: number = 10;
    public items: Deployment[] = [];
    public params: DeploymentsFilters;
    public dataLoaded: boolean = false;
    public pagination: Pagination = {
        limit: this.limit,
        offset: 0,
        listLength: this.items.length
    };

    private subscriptions: Subscription[] = [];

    constructor(private router: Router,
                private deploymentsService: DeploymentsService,
                private globalSearchService: GlobalSearchService,
                modalService: ModalService,
                notificationsService: NotificationsService) {
        this.bulkUpdater = new BulkUpdater<Deployment>(modalService, notificationsService).
            addAction('terminate', {
                title: 'Terminate Deployments',
                confirmation: count => this.actionConfirmation(count, 'terminate'),
                execute: deployment => this.deploymentsService.deleteDeploymentById(deployment.id, false).toPromise(),
                isApplicable: deployment => this.isDeploymentInStatus(deployment, ACTIONS_BY_STATUS.TERMINATE),
                warningMessage: deployments => this.formatWarningMessage(deployments, ACTIONS_BY_STATUS.TERMINATE),
                postMessage: (successfulCount, failedCount) => this.postActionMessage(successfulCount, failedCount, 'terminated')
            }).addAction('stop', {
                title: 'Stop Deployments',
                confirmation: count => this.actionConfirmation(count, 'stop'),
                execute: deployment => this.deploymentsService.stopDeployment(deployment.id, false).toPromise(),
                isApplicable: deployment => this.isDeploymentInStatus(deployment, ACTIONS_BY_STATUS.STOP),
                warningMessage: deployments => this.formatWarningMessage(deployments, ACTIONS_BY_STATUS.STOP),
                postMessage: (successfulCount, failedCount) => this.postActionMessage(successfulCount, failedCount, 'stopped')
            }).addAction('start', {
                title: 'Start Deployments',
                confirmation: count => this.actionConfirmation(count, 'start'),
                execute: deployment => this.deploymentsService.startDeployment(deployment.id, false).toPromise(),
                isApplicable: deployment => this.isDeploymentInStatus(deployment, ACTIONS_BY_STATUS.START),
                warningMessage: deployments => this.formatWarningMessage(deployments, ACTIONS_BY_STATUS.START),
                postMessage: (successfulCount, failedCount) => this.postActionMessage(successfulCount, failedCount, 'started')
            });
        this.bulkUpdater.actionExecuted.subscribe(action => this.updateItems(this.params, this.pagination, true));
    }

    private postActionMessage(successfullCount: number, failedCount: number, verb: string) {
        let beVerb = successfullCount === 1 ? 'was' : 'were';
        let entity = successfullCount === 1 ? 'deployment' : 'deployments';
        return `${successfullCount} ${entity} ${beVerb} successfully ${verb}`;
    }

    private actionConfirmation(count: number, verb: string) {
        return `Are you sure you want to ${verb} ${count} Deployment${count > 1 ? 's' : ''}?`;
    }

    private isDeploymentInStatus(deployment: Deployment, statuses: string[]) {
        return statuses.indexOf(deployment.status) > -1;
    }

    private formatWarningMessage(deployments: Deployment[], statuses: string[]) {
        return `The action is applicable only for deployments in status ${ statuses.join(',') } and will not be performed on ` +
               `${ deployments.length } deployment${ deployments.length > 1 ? 's' : ''}`;
    }

    public ngOnChanges() {
        this.params = {
            application_statuses: this.filters.application_statuses,
            app_name: this.filters['app_name'],
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
        this.globalSearchService.navigate([`/app/applications/details/${item.app_generation}/deployment/${item.id}`]);
    }

    public navigateToApp(deployment: Deployment) {
        this.router.navigate([`/app/applications/details/${deployment.app_generation}`]);
    }

    public getActivePercentValue(deployment: Deployment) {
        if (deployment.instances.length === 0) {
            return 0;
        }
        let c = deployment.instances.filter((instance: any) => instance.phase === POD_PHASE.RUNNING).length / deployment.instances.length;
        return c * 100;
    }

    private updateItems(params: any, pagination: Pagination, hideLoader?: boolean) {
        this.dataLoaded = false;
        pagination.limit += 1;
        this.subscriptions.push(this.getDeployments(params, pagination, hideLoader).subscribe(result => {
            this.dataLoaded = true;

            this.items = result.slice(0, this.limit) || [];

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

    private getDeployments(params: any, pagination: Pagination, hideLoader?: boolean) {
        let parameters = {
            limit: null,
            offset: null,
            app_name: null,
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

        if (params.app_name) {
            parameters.app_name = params.app_name;
        }

        if (params.application_statuses && params.application_statuses.length) {
            parameters.status = params.application_statuses;
        }

        return this.deploymentsService.getDeployments(parameters, hideLoader);
    }
}
