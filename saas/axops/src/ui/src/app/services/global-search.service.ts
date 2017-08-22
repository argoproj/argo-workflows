import { Inject, Injectable, Output, EventEmitter } from '@angular/core';
import { NavigationExtras, Router } from '@angular/router';
import { Location } from '@angular/common';
import { Http, URLSearchParams, Headers } from '@angular/http';
import { Observable } from 'rxjs';

import { SearchIndex } from '../model';

@Injectable()
export class GlobalSearchService {
    @Output() toggleGlobalSearch: EventEmitter<boolean> = new EventEmitter();

    private backToSearchUrl: string;

    constructor(@Inject(Http) private http, private router: Router, private location: Location) {
    }

    popBackToSearchUrl(): string {
        let backToSearchUrl =  this.backToSearchUrl;
        this.backToSearchUrl = null;
        return backToSearchUrl;
    }

    navigate(commands: any[], extras?: NavigationExtras): void {
        this.backToSearchUrl = this.location.path();
        this.toggleGlobalSearch.emit(false);
        this.router.navigate(commands, extras);
    }

    /**
     * Search suggestions (auto-complete)
     */
    getSuggestions(params?: {
        type?: string,
        key?: string[],
        search?: string,
        search_fields?: string[], }, hideLoader?: boolean): Observable<SearchIndex[]> {
        let filter = new URLSearchParams();
        let headers = new Headers();

        if (params.type) {
            filter.set('type', params.type.toString());
        }
        if (params.key) {
            filter.set('key', params.key.toString());
        }
        if (params.search) {
            filter.set('search', params.search.toString());
        }
        if (params.search_fields) {
            filter.set('search_fields', params.search_fields.toString());
        }

        if (hideLoader) {
            headers.append('isUpdated', hideLoader.toString());
        }

        return this.http.get(`v1/search/indexes`, { headers: headers, search: filter})
            .map(res => res.json().data.map(item => new SearchIndex(item)));
    }
}
