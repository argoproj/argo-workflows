import { Http } from '@angular/http';
import { Injectable } from '@angular/core';
import { Router, NavigationEnd } from '@angular/router';

import { User } from '../model';

import { SystemService } from './system.service';

let initializationPromise: Promise<any> = null;

const PORTAL_URL = 'https://portal.applatix.com/api';

@Injectable()
export class TrackingService {

    constructor(private router: Router, private systemService: SystemService, private http: Http) {
    }

    public initialize(user: User): Promise<any> {
        if (!initializationPromise) {
            initializationPromise = this.doInitialize(user);
        }
        return initializationPromise;
    }

    public async sendUsageEvent(type: string, email?: string, details?: any) {
        let versionInfo = await this.systemService.getVersion().toPromise();
        let body: any = { type, clusterId: versionInfo.cluster_id };
        if (email) {
            body.email = email;
        }
        if (details) {
            body.details = details;
        }
        this.http.post(`${PORTAL_URL}/cluster-usage-events`, body).toPromise();
    }

    private async trackInstallation() {
        let installReported = await this.systemService.getClusterSetting('install-reported');
        if (!installReported) {
            await this.sendUsageEvent('install');
            await this.systemService.createClusterSetting('install-reported', 'true');
        }
    }

    private loadGa(): Promise<any> {
        return new Promise(resolve => {
            let script = require('scriptjs');
            script('https://www.google-analytics.com/analytics.js', () => {
                resolve(window['ga']);
            });
        });
    }

    private doInitialize(user: User): Promise<any> {
        let promises = [];
        promises.push(this.systemService.getClusterSetting('ax-ga-id').catch(e => null).then(gaId => {
            if (gaId) {
                return this.loadGa().then(ga => {
                    ga('create', gaId, 'auto');
                    ga('set', 'userId', user.username);
                    this.router.events.subscribe(event => {
                        if (event instanceof NavigationEnd) {
                            ga('send', 'pageview', event.url);
                        };
                    });
                });
            } else {
                return null;
            }
        }));
        if (user.isAdmin()) {
            promises.push(this.trackInstallation());
        }
        return Promise.all(promises);
    }
}
