import { Component, Input, Output, EventEmitter, OnDestroy } from '@angular/core';

import { Observable, Observer, Subscription } from 'rxjs';

import { NotificationsService } from 'argo-ui-lib/src/components';

import { Deployment, Volume } from '../../../model';
import { DeploymentsService, VolumesService } from '../../../services';
import { Router } from '@angular/router';

@Component({
    selector: 'ax-deployment-details',
    templateUrl: './deployment-details.html',
    styles: [ require('./deployment-details.scss') ],
})
export class DeploymentDetailsComponent implements OnDestroy {

    private deploymentChangedObserver: Observer<any>;
    private deploymentChangedSubscription: Subscription;

    @Input()
    set deployment(value: Deployment) {
        if (value) {
            this.currentDeployment = value;
            if (this.deploymentChangedObserver) {
                this.deploymentChangedObserver.next({});
            }
        }
    };

    @Output()
    public onClose = new EventEmitter();
    @Output()
    public onShowHistory = new EventEmitter();
    @Output()
    public onShowHistoryDetails = new EventEmitter<{ id: string }>();
    @Output()
    public onRedeploy = new EventEmitter();

    public instances: number = 0;
    public activeEditScale: boolean;
    public currentDeployment: Deployment;
    public volumes: Volume | { chartData: {y: number}[] }[] = [];

    public chartOptions = {
        chart: {
            type: 'pieChart',
            height: 150,
            margin: { top: 0, left: 0, right: 0, bottom: 0 },
            showLabels: false,
            duration: 500,
            labelThreshold: 0.01,
            labelSunbeamLayout: true,
            showLegend: false,
            donut: true,
            donutRatio: 0.70,
            tooltip: { enabled: false },
            color: [
                // used color
                '#00A2B3',
                // not used
                '#CCD6DD'
            ]
        }
    };

    constructor(
            private router: Router, private deploymentsService: DeploymentsService, private notificationsService: NotificationsService, private volumesService: VolumesService) {
        this.deploymentChangedSubscription = Observable.create(observer => {
            this.deploymentChangedObserver = observer;
        }).bufferTime(500).subscribe(events => {
            if (events.length > 0) {
                this.refreshDeploymentInfo();
            }
        });
    }

    public ngOnDestroy() {
        if (this.deploymentChangedSubscription) {
            this.deploymentChangedSubscription.unsubscribe();
            this.deploymentChangedSubscription = null;
        }
    }

    public redeploy() {
        this.onRedeploy.emit(this.currentDeployment);
    }

    public onClosePanel() {
        this.onClose.emit();
    }

    public navigateToTemplate(templateId: string) {
        this.router.navigate([`/app/timeline/jobs/${templateId}`]);
    }

    public addInstance() {
        if (!this.activeEditScale) {
            this.instances = this.currentDeployment.instances.length;
        }
        this.activeEditScale = true;
        this.instances += 1;
    }

    public subtractInstance() {
        if (!this.activeEditScale) {
            this.instances = this.currentDeployment.instances.length;
        }
        this.activeEditScale = true;
        if (this.instances > 0) {
            this.instances -= 1;
        }
    }

    public editInstances() {
        this.deploymentsService.scaleDeployment(this.currentDeployment.id, this.instances).subscribe(() => {
            this.notificationsService.success('Deployed scale updated. Change will reflect shortly.');
            this.activeEditScale = false;
        });
    }

    public cancelEditInstances() {
        this.instances = this.currentDeployment.instances.length;
        this.activeEditScale = false;
    }

    public navigateToRevisionHistory() {
        this.onShowHistory.emit( {appId: this.currentDeployment.app_id} );
    }

    private async refreshDeploymentInfo() {
        this.volumesService.getVolumes({ deployment_id: this.currentDeployment.deployment_id }).then(volumes => {
            this.volumes = volumes.map(vol => Object.assign(vol, { chartData: [{y: vol.usagePercentage}, {y: 100 - parseInt(vol.usagePercentage, 10)}] }));
        });
    }
}
