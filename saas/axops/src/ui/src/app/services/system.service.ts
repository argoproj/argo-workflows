import { Observable } from 'rxjs';
import { Injectable } from '@angular/core';
import { Http, Headers, URLSearchParams } from '@angular/http';
import { VersionInfo } from '../model';
import { AxHeaders } from './headers';

@Injectable()
export class SystemService {
    constructor(private http: Http) {
    }

    getDnsName() {
        return this.http.get(`v1/system/settings/dnsname`)
            .map(res => res.json());
    }

    updateDnsName(dnsName: { 'dnsname': string }) {
        return this.http.put(`v1/system/settings/dnsname`, JSON.stringify(dnsName))
            .map(res => res.json());
    }

    getVersion(): Observable<VersionInfo> {
        let customHeader = new Headers();
        customHeader.append('isUpdated', 'true');
        return this.http.get(`v1/system/version`, { headers: customHeader })
            .map(res => res.json());
    }

    getSpotInstanceConfig() {
        return this.http.get(`v1/system/settings/spot_instance_config`)
            .map(res => res.json());
    }

    updateSpotInstanceConfig(option: { 'asgs': 'none' | 'partial' | 'all'}, hideLoader?: boolean) {
        let customHeader = new Headers();
        if (hideLoader) {
            customHeader.append('isUpdated', hideLoader.toString());
        }
        return this.http.put(`v1/system/settings/spot_instance_config`,
            JSON.stringify(option), { headers: customHeader })
            .map(res => res.json());
    }

    isPlayground(): Promise<boolean> {
        return this.http.get('v1/sandbox/status', { headers: new AxHeaders({ noErrorHandling: true, noLoader: true }) }).toPromise().then(
            res => res.json().enabled).catch(() => false);
    }

    getClusterSetting(key: string): Promise<string> {
        let search = new URLSearchParams();
        search.set('key', key);
        return this.http.get('v1/cluster/settings', { search, headers: new AxHeaders({ noErrorHandling: true }) }).toPromise().then(res => {
            let items = res.json().data;
            if (items.length > 0) {
                return items[0].value;
            } else {
                return null;
            }
        });
    }

    getAccessSettings(): Promise<{ trusted_cidrs: string[] }> {
        return this.http.get('v1/system/settings/security_groups_config').map(res => <{ trusted_cidrs: string[] }> res.json()).toPromise();
    }

    updateAccessSettings(settings: { trusted_cidrs: string[] }): Promise<any> {
        return this.http.put('v1/system/settings/security_groups_config', settings).map(res => <{ trusted_cidrs: string[] }> res.json()).toPromise();
    }
}
