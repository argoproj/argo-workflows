import * as _ from 'lodash';
import {Inject, Injectable} from '@angular/core';
import { Http, URLSearchParams, Headers } from '@angular/http';
import {Observable} from 'rxjs';
import {CustomView} from '../model';

@Injectable()
export class CustomViewService {
      constructor(@Inject(Http) private http: Http) {
    }

    /**
     * Retrieves all custom views in the system
     */
    getCustomViews(params?: {
        id?: string,
        name?: string
        type?: string
        user_id?: string
        username?: string
        limit?: string
        search?: string
    }, hideLoader?: boolean): Observable<CustomView[]> {
        let urlParams = new URLSearchParams();
        let headers = new Headers();

        if (hideLoader) {
            headers.append('isUpdated', hideLoader.toString());
        }
        if (params) {
            if (params.id) {
                urlParams.set('id', params.id);
            }
            if (params.name) {
                urlParams.set('name', params.name);
            }
            if (params.type) {
                urlParams.set('type', params.type);
            }
            if (params.user_id) {
                urlParams.set('user_id', params.user_id);
            }
            if (params.username) {
                urlParams.set('username', params.username);
            }
            if (params.limit) {
                urlParams.set('limit', params.limit);
            }
            if (params.search) {
                urlParams.set('search', params.search);
            }
        }
        return this.http.get(`v1/custom_views?${urlParams.toString()}`, { headers: headers })
            .map(res => _.map(res.json().data, item => new CustomView(item)));
    }

    /**
     * Get specific custom view
     */
    getCustomViewById(customViewId): Observable<CustomView> {
        return this.http.get(`v1/custom_views/${customViewId}`)
            .map(res => {
                return new CustomView(res.json());
            });
    }

    /**
     * Remove a custom view
     */
    deleteCustomView(id: string, hideLoader?: boolean) {
        let headers = new Headers();
        if (hideLoader) {
            headers.append('isUpdated', hideLoader.toString());
        }

        return this.http.delete(`v1/custom_views/${id}`, { headers: headers })
            .map(res => res.json());
    }

    /**
     * Creates an instance of Custom View
     */
    createCustomView(customView: CustomView): Observable<CustomView> {
        return this.http.post(`v1/custom_views`, JSON.stringify(customView))
            .map(res => new CustomView(res.json()));
    }

    /**
     * Update an instance of custom view
     */
    updateCustomView(customView: CustomView): Observable<CustomView> {
        return this.http.put(`v1/custom_views/${customView.id}`, JSON.stringify(customView))
            .map(res => new CustomView(res.json()));
    }
}
