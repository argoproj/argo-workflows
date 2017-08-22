import { Component, Input, Output, EventEmitter } from '@angular/core';

import { DropdownMenuSettings } from 'argo-ui-lib/src/components';

import { Application, Deployment, Pod, APPLICATION_STATUSES } from '../../../model';
import { DeploymentsService } from '../../../services';
import { STATUS_FILTERS } from '../view-models';

@Component({
    selector: 'ax-app-graph',
    templateUrl: './app-graph.html',
    styles: [require('./app-graph.scss')],
})
export class AppGraphComponent {

    public statuses = APPLICATION_STATUSES;
    public deployments: Deployment[]  | { noDeployments: boolean }[];

    public get application(): Application {
        return this.appInfo;
    }

    @Input()
    public set application(appInfo: Application) {
        if (appInfo) {
            this.appInfo = appInfo;
            this.refreshAppDeployments();
        } else {
            this.appInfo = null;
            this.deployments = [];
        }
    }

    @Input()
    public selectedDeploymentName: string;

    @Input()
    public podMenuCreator: (deployment: Deployment, pod: Pod) => DropdownMenuSettings;

    @Input()
    public applicationMenuCreator: (application: Application) => DropdownMenuSettings;

    @Input()
    public deploymentMenuCreator: (deployment: Deployment) => DropdownMenuSettings;

    @Input()
    public set statusFilters(statusFiltersInfo: string[]) {
        this.statusFiltersInfo = statusFiltersInfo;
        this.refreshAppDeployments();
    }

    @Output()
    public deploymentSelected = new EventEmitter<string>();

    @Output()
    public podSelected = new EventEmitter<{deployment: string, pod: string}>();

    private appInfo: Application;
    private statusFiltersInfo: string[] = [];

    constructor(private deploymentsService: DeploymentsService) {}

    public trackByDeploymentName(deployment: Deployment) {
        return deployment.name;
    }

    public trackByPodName(pod: Pod) {
        return pod.name;
    }

    private refreshAppDeployments() {
        this.deployments = this.appInfo ? Array.from(this.appInfo.deployments.slice()).sort((first, second) => first.name.localeCompare(second.name)) : [];
        if (this.statusFiltersInfo && this.statusFiltersInfo.length > 0) {
            this.deployments = this.deployments.filter(item => {
                return !!this.statusFiltersInfo.find(filterKey => STATUS_FILTERS[filterKey].statuses.indexOf(item.status) > -1);
            });
        }
        if (this.deployments.length === 0) {
            this.deployments = [ { noDeployments: true} ];
        }
    }
}

@Component({
    selector: 'ax-app-graph-status-icon',
    template: `<span [ax-tooltip]="statusMessage || status || ''">
        <i *ngIf="status === statuses.UPGRADING || status === statuses.TERMINATING || status === statuses. INIT || status === statuses.WAITING"
            class="fa fa-circle-o-notch status-icon--running status-icon--spin app-graph__node-icon app-graph__node-icon--progress app-graph__node-icon--status"></i>
        <i *ngIf="status === statuses.STOPPED" class="ax-icon-stop app-graph__node-icon--stop app-graph__node-icon--status"></i>
        <i *ngIf="status === statuses.TERMINATED" class="ax-icon-terminate app-graph__node-icon--stop app-graph__node-icon--status"></i>
        <i *ngIf="status === statuses.ERROR" class="fa fa-circle-o app-graph__node-icon--stop app-graph__node-icon--status"></i>
    </span>`
})
export class AppGraphStatusIconComponent {
    public statuses = APPLICATION_STATUSES;

    @Input()
    public status: string;
    @Input()
    public statusMessage: string;
}
