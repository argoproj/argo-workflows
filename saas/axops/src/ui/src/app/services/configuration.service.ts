import { Injectable } from '@angular/core';
import { Http, URLSearchParams } from '@angular/http';
import { Configuration } from '../model';
import { AxHeaders } from './headers';

@Injectable()
export class ConfigsService {
    constructor(private http: Http) {
    }

    public getConfigurations(params: { user?: string, name?: string } = {}, noLoader = false): Promise<Configuration[]> {
        let url = 'v1/configurations';

        let search = new URLSearchParams();

        if (params.user) {
            search.set('user', params.user);
        }
        if (params.name) {
            search.set('name', params.name);
        }
        return this.http.get(url, { search, headers: new AxHeaders({ noLoader }) }).toPromise().then(res => <Configuration[]>res.json());
    }

    public createConfiguration(config: Configuration): Promise<any> {
        let search = new URLSearchParams();
        search.set('description', config.description || '');
        return this.http.post(`v1/configurations/${config.user}/${config.name}`, config.value || {}, {search}).toPromise();
    }

    public updateConfiguration(config: Configuration): Promise<any> {
        let search = new URLSearchParams();
        search.set('description', config.description || '');
        return this.http.put(`v1/configurations/${config.user}/${config.name}?${search.toString()}`, config.value || {}).toPromise();
    }

    public deleteConfiguration(params: { user: string, name: string }): Promise <any> {
        return this.http.delete(`v1/configurations/${params.user}/${params.name}`).toPromise();
    }
}
