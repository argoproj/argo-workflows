import { Injectable } from '@angular/core';
import { Http, URLSearchParams } from '@angular/http';
import { Configuration } from '../model';
import { AxHeaders } from './headers';

@Injectable()
export class ConfigsService {
    constructor(private http: Http) {
    }

    public getConfigurations(params: { user?: string, name?: string} = {}, noLoader = true): Promise<Configuration[]> {
        let url = 'v1/configurations';

        let search = new URLSearchParams();

        if (params.user) {
            search.set('user', params.user);
        }
        if (params.name) {
            search.set('name', params.name);
        }
        return this.http.get(url, { search, headers: new AxHeaders({ noLoader: noLoader }) }).toPromise().then(res => <Configuration[]>res.json());
    }

    public getUserConfiguration(user: string, name: string, showSecrets, noLoader = true): Promise<Configuration> {
        let url = `v1/configurations/${user}/${name}`;

        let search = new URLSearchParams();
        if (showSecrets) {
            search.set('show_secrets', 'true');
        }
        return this.http.get(url, { search, headers: new AxHeaders({ noLoader: noLoader }) }).toPromise().then(res => <Configuration>res.json());
    }

    public createConfiguration(config: Configuration): Promise<any> {
        return this.http.post('v1/configurations', config).toPromise();
    }

    public updateConfiguration(config: Configuration): Promise<any> {
        return this.http.put(`v1/configurations/${config.user}/${config.name}`, config).toPromise();
    }

    public deleteConfiguration(params: { user: string, name: string }): Promise <any> {
        return this.http.delete(`v1/configurations/${params.user}/${params.name}`).toPromise();
    }
}
