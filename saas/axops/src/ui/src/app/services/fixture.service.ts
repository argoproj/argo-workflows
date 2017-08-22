import { Injectable } from '@angular/core';
import { Http, URLSearchParams } from '@angular/http';

import { FixtureClass, FixtureInstance, FixtureTemplate } from '../model';
import { AxHeaders } from './headers';

function deserialiseFixtureClass(data: any): FixtureClass {
    return Object.assign(new FixtureClass(), data);
}

function deserialiseFixtureInstance(data: any): FixtureInstance {
    return Object.assign(new FixtureInstance(), data);
}

@Injectable()
export class FixtureService {
    constructor(private http: Http) {
    }

    public getFixtureClasses(params = { name: <string> null }): Promise<FixtureClass[]> {
        let search = new URLSearchParams();
        if (params.name) {
            search.set('name', name);
        }
        return this.http.get('v1/fixture/classes', { search, headers: new AxHeaders({ noLoader: true })}).toPromise().
            then(res => <any[]>res.json().data).then(items => items.map(deserialiseFixtureClass));
    }

    public getFixtureClass(id: string): Promise<FixtureClass> {
        return this.http.get(`v1/fixture/classes/${id}`, {headers: new AxHeaders({ noLoader: true })}).toPromise().then(res => res.json()).then(deserialiseFixtureClass);
    }

    public createFixtureClass(templateId: string): Promise<FixtureClass> {
        return this.http.post('v1/fixture/classes', { template_id: templateId }).toPromise().then(res => res.json().data).then(deserialiseFixtureClass);
    }

    public getFixtureInstances(classId: string): Promise<FixtureInstance[]> {
        let search = new URLSearchParams();
        search.set('class_id', classId);
        return this.http.get('v1/fixture/instances', { search, headers: new AxHeaders({ noLoader: true }) }).toPromise().
            then(res => <any[]>res.json().data).then(items => items.map(deserialiseFixtureInstance));
    }

    public getFixtureInstance(id: string): Promise<FixtureInstance> {
        return this.http.get(`v1/fixture/instances/${id}`, {headers: new AxHeaders({ noLoader: true })}).toPromise().then(res => res.json()).then(deserialiseFixtureInstance);
    }

    public updateFixtureInstance(fixtureInstance: FixtureInstance): Promise<FixtureInstance> {
        return this.http.put(`v1/fixture/instances/${fixtureInstance.id}`, fixtureInstance).toPromise().then(res => res.json()).then(deserialiseFixtureInstance);
    }

    public createFixtureInstance(instance: FixtureInstance): Promise<FixtureInstance> {
        return this.http.post('v1/fixture/instances', instance).toPromise().then(res => res.json()).then(deserialiseFixtureInstance);
    }

    public updateFixtureClass(id: string, templateId: string): Promise<FixtureClass> {
        return this.http.put(`v1/fixture/classes/${id}`, { template_id: templateId }).toPromise().then(res => res.json()).then(deserialiseFixtureClass);
    }

    public getFixtureTemplates(): Promise<FixtureTemplate[]> {
        return this.http.get('v1/fixture/templates', {headers: new AxHeaders({ noLoader: true })}).toPromise().then(res => res.json().data);
    }

    public runFixtureInstanceAction(fixtureInstanceId: string, action: string, parameters?: any): Promise<any> {
        let data = { action };
        if (parameters) {
            data['parameters'] = parameters;
        }
        return this.http.post(`v1/fixture/instances/${fixtureInstanceId}/action`, data).toPromise();
    }

    public async setFixtureInstanceEnabled(fixtureInstanceId: string, isEnabled: boolean): Promise<any> {
        await this.updateFixtureInstance({ id: fixtureInstanceId, enabled: isEnabled });
    }

    public async setFixtureStatus(fixtureInstanceId: string, status: string): Promise<any> {
        await this.updateFixtureInstance({ id: fixtureInstanceId, status: status });
    }

    public deleteFixtureInstance(fixtureInstanceId: string): Promise<any> {
        return this.http.delete(`v1/fixture/instances/${fixtureInstanceId}`, null).toPromise();
    }

    public deleteFixtureClass(fixtureClassId: string): Promise<any> {
        return this.http.delete(`v1/fixture/classes/${fixtureClassId}`, null).toPromise();
    }

    public getUsageStats(): Promise<{[name: string]: {available: number, total: number} }> {
        return this.http.get('v1/fixture/summary?group_by=class_name', {headers: new AxHeaders({ noLoader: true })}).toPromise().then(res => res.json());
    }
}
