import * as _ from 'lodash';
import { Component, OnInit, OnDestroy } from '@angular/core';
import { Observable, Subscription } from 'rxjs';

import { Host } from '../../model';
import { HostService } from '../../services';
import { HasLayoutSettings, LayoutSettings } from '../layout';

@Component({
    templateUrl: './hosts.html',
    styles: [ require('./hosts.scss') ],
})
export class HostsComponent implements OnInit, OnDestroy, HasLayoutSettings {

    private hosts: Host[];
    private interval: Subscription;

    constructor(private hostService: HostService) {
    }

    ngOnInit() {
        this.getHostsAsync();
        this.getHostsInterval(5000);
    }

    ngOnDestroy() {
        this.interval.unsubscribe();
    }

    get layoutSettings(): LayoutSettings {
        return {
            pageTitle: 'Hosts',
        };
    }

    getHostsAsync(isUpdated = false) {
        this.hostService.getHostsAsync(isUpdated).subscribe(result => {
            this.hosts = _.map(result.data, (host: Host) => {
                let localHost = new Host();
                localHost.name = host.name;
                localHost.cpu = host.cpu;
                localHost.mem = host.mem;
                localHost.services = host.services;
                localHost.usage = {
                    host_id: host.usage.host_id ? host.usage.host_id : '',
                    host_name: host.usage.host_name ? host.usage.host_name : '',
                    cpu: host.usage.cpu ? host.usage.cpu : 0,
                    cpu_used: host.usage.cpu_used ? host.usage.cpu_used : 0,
                    cpu_total: host.usage.cpu_total ? host.usage.cpu_total : 0,
                    cpu_percent: host.usage.cpu_percent ? host.usage.cpu_percent : 0,
                    mem: host.usage.mem ? host.usage.mem : 0,
                    mem_percent: host.usage.mem_percent ? host.usage.mem_percent : 0
                };

                return localHost;
            });
        });
    }

    getHostsInterval(intervalInMilliseconds) {
        this.interval = Observable.interval(intervalInMilliseconds).subscribe(success => {
            if (!document.hidden) {
                this.getHostsAsync(true);
            }
        });
    }
}
