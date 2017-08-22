import { Component, OnInit } from '@angular/core';
import { SystemService } from '../../../../services';
import { NotificationsService } from 'argo-ui-lib/src/components';

@Component({
    selector: 'ax-spot-instances-panel',
    templateUrl: './spot-instances-panel.html',
})
export class SpotInstancesPanelComponent implements OnInit {
    spotInstancesEnabled: boolean = false;
    dataLoaded: boolean = true;

    constructor(
        private systemService: SystemService,
        private notificationsService: NotificationsService) {
    }

    ngOnInit() {
        this.systemService.getSpotInstanceConfig().subscribe(res => {
            this.spotInstancesEnabled = res.status === 'True';
            this.dataLoaded = true;
        },
            () => this.dataLoaded = true);
    }

    onChangeStatus(e: any): void {
        this.systemService.updateSpotInstanceConfig({'enabled': e.target.checked}, true)
            .subscribe(() => {
                this.spotInstancesEnabled = e.target.checked;
                this.notificationsService.success(`Spot Instances ${e.target.checked ? 'enabled' : 'disabled'}`);
            }, () => e.target.checked = this.spotInstancesEnabled);
    }
}
