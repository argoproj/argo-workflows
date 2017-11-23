import * as moment from 'moment';
import { HttpClient, HttpParams } from '@angular/common/http';
import { Inject, Injectable } from '@angular/core';
import { Observable } from 'rxjs/Observable';

import { Commit } from '../models';
import { AxHeaders } from './headers';

@Injectable()
export class CommitsService {
    constructor(@Inject(HttpClient) private _http) {
    }

    /**
     * Appends commits page to already loaded list of commits and remove duplicates from loaded page. Duplicates are possible because pagination is based on commit date, so
     * API might return same commits at page boundary.
     */
    addCommitsPage(items: Commit[], nextPage: Commit[]): Commit[] {
        nextPage = nextPage.slice();
        if (items.length > 0) {
            let lastIndex = items.length - 1;
            let boundaryDate = items[lastIndex].date;
            while (lastIndex > -1 && items[lastIndex].date === boundaryDate) {
                let last = items[lastIndex];
                let duplicateIndex = nextPage.findIndex(item => item.revision === last.revision);
                if (duplicateIndex > -1) {
                    nextPage.splice(duplicateIndex, 1);
                }
                lastIndex -= 1;
            }
        }
        return items.concat(nextPage);
    }

    getCommitsAsync(params?: {
        author?: string,
        committer?: string,
        repo?: string,
        revision?: string,
        branch?: string,
        minTime?: moment.Moment,
        maxTime?: moment.Moment,
        search?: string,
        searchFields?: string[],
        limit?: number,
        offset?: number,
        sort?: string,
        repo_branch?: {[name: string]: string[]},
    }, hideLoader = true): Observable<{data: Commit[]}> {
        let search = new HttpParams();

        if (params.repo) {
            search.set('repo', params.repo.toString());
        }

        if (params.revision) {
            search.set('revision', params.revision.toString());
        }

        if (params.branch) {
            search.set('branch', params.branch.toString());
        }

        if (params.author) {
            search.set('author', params.author.toString());
        }

        if (params.committer) {
            search.set('committer', params.committer.toString());
        }

        if (params.minTime) {
            search.set('min_time', params.minTime.unix().toString());
        }

        if (params.maxTime) {
            search.set('max_time', params.maxTime.unix().toString());
        }

        if (params.search) {
            search.set('search', '~' + params.search.toString());
        }

        if (params.limit) {
            search.set('limit', params.limit.toString());
        }

        if (params.offset) {
            search.set('offset', params.offset.toString());
        }

        if (params.sort) {
            search.set('sort', params.sort.toString());
        }

        if (params.repo_branch && Object.keys(params.repo_branch).length > 0) {
            search.set('repo_branch', JSON.stringify(params.repo_branch));
        }

        if (params.searchFields) {
            search.set('search_fields', params.searchFields.join(','));
        }

        return this._http.get('v1/commits', {headers: new AxHeaders({ noLoader: hideLoader }), search: search})
            .map(res => res.json());
    }

    getCommitByRevision(revisionId: string, hideLoader = true): Observable<Commit> {
        return this._http.get(`v1/commits/${revisionId}`, {headers: new AxHeaders({ noLoader: hideLoader })})
            .map(res => res.json());
    }
}
