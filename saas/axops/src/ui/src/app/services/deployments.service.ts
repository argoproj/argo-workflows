import * as _ from 'lodash';
import { Injectable } from '@angular/core';
import { Http, URLSearchParams, Headers } from '@angular/http';
import { Deployment } from '../model';
import { Observable } from 'rxjs';
import { HttpService } from './http.service';

import { AxHeaders } from './headers';

@Injectable()
export class DeploymentsService {

    constructor(private http: Http, private httpService: HttpService) {
    }

    getDeployments(params?: {
        name?: string,
        description?: string,
        status?: string,
        template_id?: string,
        task_id?: string,
        app_generation?: string,
        app_id?: string,
        app_name?: string,
        sort?: string,
        search?: string,
        fields?: string,
        limit?: number,
        offset?: number,
    }, hideLoader?: boolean): Observable<Deployment[]> {
        let filter = new URLSearchParams();
        let headers = new Headers();

        if (params.name) {
            filter.set('name', params.name.toString());
        }
        if (params.description) {
            filter.set('description', params.description.toString());
        }
        if (params.status) {
            filter.set('status', params.status.toString());
        }
        if (params.template_id) {
            filter.set('template_id', params.template_id.toString());
        }
        if (params.task_id) {
            filter.set('task_id', params.task_id.toString());
        }
        if (params.app_generation) {
            filter.set('app_generation', params.app_generation.toString());
        }
        if (params.app_id) {
            filter.set('app_id', params.app_id.toString());
        }
        if (params.app_name) {
            filter.set('app_name', params.app_name.toString());
        }
        if (params.sort) {
            filter.set('sort', params.sort.toString());
        }
        if (params.search) {
            filter.set('search', params.search.toString());
        }
        if (params.fields) {
            filter.set('fields', params.fields.toString());
        }
        if (params.limit) {
            filter.set('limit', params.limit.toString());
        }
        if (params.offset) {
            filter.set('offset', params.offset.toString());
        }

        if (hideLoader) {
            headers.append('isUpdated', hideLoader.toString());
        }

        return this.http.get(`v1/deployments?${filter.toString()}`, { headers: headers })
            .map((res) => {
                return _.map(res.json().data, (item) => { return new Deployment(item); });
            });
    }

    public getDeploymentById(id: string, hideLoader: boolean = true): Observable<Deployment> {
        return this.http.get(`v1/deployments/${id}`, {headers: new AxHeaders({noLoader: hideLoader})})
            .map((res) => {
                return new Deployment(res.json());
            });
    }

    public getDeploymentHistory(params?: {
        appName: string,
        deploymentName: string,
        limit?: number,
        offset?: number,
    }, hideLoader: boolean = true): Observable<Deployment[]> {
        let filter = new URLSearchParams();
        let headers = new Headers();

        if (hideLoader) {
            headers.append('isUpdated', hideLoader.toString());
        }

        if (params.deploymentName) {
            filter.set('name', params.deploymentName.toString());
        }
        if (params.appName) {
            filter.set('app_name', params.appName.toString());
        }
        if (params.limit) {
            filter.set('limit', params.limit.toString());
        }
        if (params.offset) {
            filter.set('offset', params.offset.toString());
        }

        return this.http.get(`v1/deployment/histories?${filter.toString()}`, { headers: headers }).map(res => res.json().data);
    }

    public deleteDeploymentById(id: string, hideLoader?: boolean) {
        let headers = new Headers();
        if (hideLoader) {
            headers.append('isUpdated', hideLoader.toString());
        }
        return this.http.delete(`v1/deployments/${id}`, { headers: headers }).map(res => res.json());
    }

    public startDeployment(id: string, hideLoader?: boolean) {
        let headers = new Headers();
        if (hideLoader) {
            headers.append('isUpdated', hideLoader.toString());
        }
        return this.http.post(`v1/deployments/${id}/start`, {}, { headers: headers }).map(res => res.json());
    }

    public stopDeployment(id: string, hideLoader?: boolean) {
        let headers = new Headers();
        if (hideLoader) {
            headers.append('isUpdated', hideLoader.toString());
        }
        return this.http.post(`v1/deployments/${id}/stop`, {}, { headers: headers }).map(res => res.json());
    }

    public scaleDeployment(id: string, scale: number, hideLoader?: boolean): Observable<any> {
        let headers = new Headers();
        if (hideLoader) {
            headers.append('isUpdated', hideLoader.toString());
        }
        return this.http.post(`v1/deployments/${id}/scale`, { min: scale }, { headers: headers }).map(res => res.json());
    }

    public getContainerLiveLog(deploymentId: string, instance: string, container: string) {
        let url = `v1/deployments/${deploymentId}/livelog?instance=${instance}&container=${container}`;
        return this.httpService.loadEventSource(url).map(data => JSON.parse(data).log);
    }

    public getServiceStepEvents(id: string): Observable<any> {
        let query = new URLSearchParams();
        query.set('id', id);
        return this.httpService.loadEventSource(`v1/deployment/events?${query.toString()}`).map(data => JSON.parse(data));
    }
}
