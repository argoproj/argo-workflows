import { Component, Input, Output, EventEmitter } from '@angular/core';

import { Deployment } from '../../../model';
import { DeploymentsService } from '../../../services';

@Component({
    selector: 'ax-deployment-history',
    templateUrl: './deployment-history.html',
    styles: [ require('./deployment-history.scss') ],
})
export class DeploymentHistoryComponent {

    private deploymentInfo: Deployment;
    protected readonly limit: number = 10;

    public get deployment(): Deployment {
        return this.deploymentInfo;
    }

    @Input()
    public set deployment(val: Deployment) {
        this.offset = 0;
        this.canScroll = false;
        this.deploymentInfo = val;
        if (this.deploymentInfo) {
            this.getDeploymentHistory();
        }
    }

    @Output()
    public onClose: EventEmitter<any> = new EventEmitter();

    @Output()
    public onShowHistoryDetails = new EventEmitter<{ id: string }>();

    public deployments: Deployment[] = [];
    public offset: number = 0;
    public onScrollLoading: boolean = false;
    public canScroll: boolean = false;

    constructor(private deploymentsService: DeploymentsService) {}

    private getDeploymentHistory() {
        this.onScrollLoading = true;
        this.deploymentsService.getDeploymentHistory({
            appName: this.deployment.app_name,
            deploymentName: this.deployment.name,
            limit: this.limit,
            offset: this.offset
        }).toPromise().then(res => {
            if (this.offset === 0) {
                this.deployments = [this.deploymentInfo];
            }
            this.deployments = this.deployments.concat(res);
            this.offset += (res || []).length;
            this.onScrollLoading = false;
            this.canScroll = (res || []).length === this.limit;
        });
    }

    public onClosePanel() {
        this.onClose.emit();
    }

    public onScroll() {
        if (this.canScroll && !this.onScrollLoading) {
            this.getDeploymentHistory();
        }
    }

    public trackByDeploymentId(deployment: Deployment) {
        return deployment.id;
    }
}
