import { Component, Input } from '@angular/core';

import { Deployment } from '../../../model';
import { DeploymentsService } from '../../../services';

@Component({
    selector: 'ax-deployment-history-details',
    templateUrl: './deployment-history-details.html',
    styles: [ require('./deployment-history-details.scss') ]
})
export class DeploymentHistoryDetailsComponent {

    @Input()
    public set id(val: string) {
        if (val) {
            if ((this.deployment && this.deployment.id) !== val) {
                this.deploymentService.getDeploymentById(val).subscribe(res => this.deployment = res);
            }
        } else {
            this.deployment = null;
        }
    }

    public selectedTabKey = 'summary';
    public deployment: Deployment;

    constructor(private deploymentService: DeploymentsService) {

    }
}
