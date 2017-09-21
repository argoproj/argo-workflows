import { Component, OnDestroy, OnInit } from '@angular/core';
import { Subscription } from 'rxjs/Subscription';

import { SystemService } from '../../../../services';
import { NotificationsService } from 'argo-ui-lib/src/components';

@Component({
    selector: 'ax-spot-instances-panel',
    templateUrl: './spot-instances-panel.html',
    styles: [ require('./spot-instances-panel.scss') ]
})
export class SpotInstancesPanelComponent implements OnInit, OnDestroy {
    public spotInstancesOption: 'none' | 'partial' | 'all';
    public isSpotInstanceEnabled: boolean;
    public dataLoaded: boolean = true;
    public statusChangeLoader: boolean = false;
    public optionChangeLoader: boolean = false;
    private subscription: Subscription = null;

    constructor(
        private systemService: SystemService,
        private notificationsService: NotificationsService) {
    }

    public ngOnInit() {
        this.systemService.getSpotInstanceConfig().subscribe((res: { 'spot_instances_option': 'none' | 'partial' | 'all', enabled: 'True' | 'False'}) => {
            this.spotInstancesOption = res.spot_instances_option;
            this.isSpotInstanceEnabled = res.enabled === 'True';
            this.dataLoaded = true;
        }, () => this.dataLoaded = true);
    }

    public ngOnDestroy() {
        this.unsubscribe();
    }

    public onChangeOption(option: 'none' | 'partial' | 'all'): void {
        this.unsubscribe();
        this.statusChangeLoader = true;
        this.subscription = this.systemService.updateSpotInstanceConfig({spot_instances_option: option, enabled: this.isSpotInstanceEnabled}, true)
            .subscribe(() => {
                this.statusChangeLoader = false;
                this.spotInstancesOption = option;
                this.notificationsService.success(`Spot Instances set to: ${ option === 'all' ? 'full' : option }`);
            });
    }

    public onChangeStatus(e: any): void {
        this.unsubscribe();
        this.optionChangeLoader = true;
        this.subscription = this.systemService.updateSpotInstanceConfig({spot_instances_option: this.spotInstancesOption, enabled: e.target.checked}, true)
            .subscribe(() => {
                this.optionChangeLoader = false;
                this.isSpotInstanceEnabled = e.target.checked;
                this.notificationsService.success(`Spot Instances is: ${e.target.checked ? 'enabled' : 'disabled'}`);
            }, () => e.target.checked = this.isSpotInstanceEnabled);
    }

    public unsubscribe() {
        if (this.subscription !== null) {
            this.subscription.unsubscribe();
            this.subscription = null;
        }
    }
}
