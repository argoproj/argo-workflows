import * as moment from 'moment';
import { Component, Input } from '@angular/core';

import { DropdownMenuSettings } from 'argo-ui-lib/src/components';

import { Deployment, Task, Commit } from '../../../model';
import { LaunchPanelService } from '../../../common';

@Component({
    selector: 'ax-deployment-history-cell',
    templateUrl: './deployment-history-cell.html',
    styles: [ require('./deployment-history-cell.scss') ],
})
export class DeploymentHistoryCellComponent {
    @Input()
    public set deployment(deployment: Deployment) {
        this.deploymentInfo = deployment;
        if (deployment) {
            let endTime = deployment.end_time > 0 ? moment.unix(deployment.end_time) : moment();
            this.runDuration = endTime.to(moment.unix(deployment.create_time), true);
            this.menuSettings = new DropdownMenuSettings([{
                title: this.deployment.end_time > 0 ? 'Rollback' : 'Redeploy',  iconName: '',
                action: async () => {
                    let task = Object.assign(new Task(), { template: this.deployment.template, parameters: this.deployment.parameters, template_id: this.deployment.template_id });
                    let commit = Object.assign(new Commit(), { repo: this.deployment.template.repo, branch: this.deployment.template.branch });
                    this.launchPanelService.openPanel(commit, task);
                }
            }]);
        }
    }

    public get deployment(): Deployment {
        return this.deploymentInfo;
    }

    public menuSettings: DropdownMenuSettings;

    public runDuration: string;

    private deploymentInfo: Deployment;

    constructor(private launchPanelService: LaunchPanelService) {
    }
}
