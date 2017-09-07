import { Component, OnInit } from '@angular/core';
import { SystemService } from '../../../../services';
import { NotificationsService } from 'argo-ui-lib/src/components';

@Component({
    selector: 'ax-spot-instances-panel',
    templateUrl: './spot-instances-panel.html',
    styles: [ require('./spot-instances-panel.scss') ]
})
export class SpotInstancesPanelComponent implements OnInit {
    public spotInstancesOption: 'none' | 'partial' | 'all';
    public dataLoaded: boolean = true;

    constructor(
        private systemService: SystemService,
        private notificationsService: NotificationsService) {
    }

    public ngOnInit() {
        this.systemService.getSpotInstanceConfig().subscribe((res: { 'asgs': 'none' | 'partial' | 'all'}) => {
            this.spotInstancesOption = res.asgs;
            this.dataLoaded = true;
        }, () => this.dataLoaded = true);
    }

    public onChangeOption(option: 'none' | 'partial' | 'all'): void {
        this.systemService.updateSpotInstanceConfig({'asgs': option}, true)
            .subscribe(() => {
                this.spotInstancesOption = option;
                this.notificationsService.success(`Spot Instances set to: ${ option === 'all' ? 'full' : option }`);
            });
    }
}
