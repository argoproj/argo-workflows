import { Component, OnInit } from '@angular/core';
import { SystemService } from '../../../../services';
import { NotificationsService } from 'argo-ui-lib/src/components';

@Component({
    selector: 'ax-spot-instances-panel',
    templateUrl: './spot-instances-panel.html',
    styles: [ require('./spot-instances-panel.scss') ]
})
export class SpotInstancesPanelComponent implements OnInit {
    spotInstancesOption: 'none' | 'partial' | 'all';
    dataLoaded: boolean = true;

    constructor(
        private systemService: SystemService,
        private notificationsService: NotificationsService) {
    }

    ngOnInit() {
        this.systemService.getSpotInstanceConfig().subscribe(res => {
            this.spotInstancesOption = res.status;
            this.dataLoaded = true;
        }, () => this.dataLoaded = true);
    }

    onChangeStatus(option: 'none' | 'partial' | 'all'): void {
        this.systemService.updateSpotInstanceConfig({'asgs': option}, true)
            .subscribe(() => {
                this.spotInstancesOption = option;
                this.notificationsService.success(`Spot Instances set to: ${ option === 'all' ? 'full' : option }`);
            });
    }
}
