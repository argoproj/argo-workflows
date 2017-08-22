import * as _ from 'lodash';
import { Injectable, NgZone } from '@angular/core';
import { Http, URLSearchParams, Headers } from '@angular/http';
import { Observable, Observer, Subscription } from 'rxjs';

import { AxHeaders } from './headers';

import { Application } from '../model';
import { HttpService } from './http.service';

@Injectable()
export class ApplicationsService {
    constructor(private http: Http, private httpService: HttpService, private zone: NgZone) {
    }

    /**
     * Load applications for given filters
     */
    public getApplications(params?: {
        name?: string,
        description?: string,
        status?: string,
        sort?: string,
        search?: string,
        fields?: string,
        limit?: number,
        searchFields?: string[],
        offset?: number,
        include_details?: boolean,
    }, hideLoader?: boolean): Observable<Application[]> {
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

        if (params.sort) {
            filter.set('sort', params.sort.toString());
        }

        if (params.search) {
            filter.set('search', params.search.toString());
        }

        if (params.searchFields) {
            filter.set('search_fields', params.searchFields.join(','));
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

        if (params.include_details) {
            filter.set('include_details', params.include_details.toString());
        }

        if (hideLoader) {
            headers.append('isUpdated', hideLoader.toString());
        }

        return this.http.get('v1/applications', { headers: headers, search: filter })
            .map(res => _.map(res.json().data, item => new Application(item)));
    }

    /**
     * Load application details by id
     */
    public getApplicationById(id: string, hideLoader: boolean = true): Observable<Application> {
        return this.http.get(`v1/applications/${id}`, { headers: new AxHeaders({noLoader: hideLoader}) }).map(res => new Application(res.json()));
    }

    public deleteAppById(id: string, hideLoader?: boolean) {
        let headers = new Headers();
        if (hideLoader) {
            headers.append('isUpdated', hideLoader.toString());
        }
        return this.http.delete(`v1/applications/${id}`, { headers: headers }).map(res => res.json());
    }

    public startApplication(id: string, hideLoader?: boolean) {
        let headers = new Headers();
        if (hideLoader) {
            headers.append('isUpdated', hideLoader.toString());
        }
        return this.http.post(`v1/applications/${id}/start`, {}, { headers: headers }).map(res => res.json());
    }

    public stopApplication(id: string, hideLoader?: boolean) {
        let headers = new Headers();
        if (hideLoader) {
            headers.append('isUpdated', hideLoader.toString());
        }
        return this.http.post(`v1/applications/${id}/stop`, {}, { headers: headers }).map(res => res.json());
    }

    private getApplicationEvents(id: string): Observable<any> {
        let query = new URLSearchParams();
        query.set('id', id);
        return this.httpService.loadEventSource(`v1/application/events?${query.toString()}`).map(data => JSON.parse(data));
    }

    public getApplicationUpdates(id: string, hideLoader = false): Observable<Application> {
        return Observable.create((observer: Observer<Application>) => {
            let subscription: Subscription = null;
            let reconnect = true;

            let ensureUnsubscribed = () => {
                if (subscription != null) {
                    subscription.unsubscribe();
                    subscription = null;
                }
            };

            let doLoadUpdates = (subscribeToEvents) => {
                this.getApplicationById(id, true).toPromise().then(task => {
                    this.zone.run(() => observer.next(task));
                    hideLoader = true;
                    if (subscribeToEvents) {
                        ensureUnsubscribed();
                        subscription = this.getApplicationEvents(task.id).bufferTime(500).subscribe(events => {
                            if (events.length > 0) {
                                doLoadUpdates(false);
                            }
                        }, () => reconnect && doLoadUpdates(true), () => reconnect && doLoadUpdates(true));
                    }
                });
            };

            doLoadUpdates(true);

            return () => {
                reconnect = false;
                ensureUnsubscribed();
            };
        });
    }
}
