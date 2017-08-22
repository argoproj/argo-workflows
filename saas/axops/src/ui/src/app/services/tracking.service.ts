import { Injectable } from '@angular/core';
import { Router, NavigationEnd } from '@angular/router';

import { User } from '../model';

import { SystemService } from './system.service';

let initializationPromise: Promise<any> = null;

@Injectable()
export class TrackingService {

    constructor(private router: Router, private systemService: SystemService) {
    }

    public initialize(user: User): Promise<any> {
        if (!initializationPromise) {
            initializationPromise = this.doInitialize(user);
        }
        return initializationPromise;
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
        return this.systemService.getClusterSetting('ax-ga-id').catch(e => null).then(gaId => {
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
        });
    }
}
