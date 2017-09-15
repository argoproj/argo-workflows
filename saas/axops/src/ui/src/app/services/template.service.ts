import { Inject, Injectable } from '@angular/core';
import { Http, URLSearchParams, Headers } from '@angular/http';
import { Observable } from 'rxjs/Observable';
import { Template, Branch } from '../model';
import { AxHeaders } from './headers';

@Injectable()
export class TemplateService {
    static normalizeRepoUrl(repo) {
        return repo.replace(/^(http|https):\/+/, '');
    }

    static areRepoUrlsEqual(first: string, second: string) {
        return TemplateService.normalizeRepoUrl(first) === TemplateService.normalizeRepoUrl(second);
    }

    constructor( @Inject(Http) private _http: Http) {
    }

    getTemplatesAsync(params?: {
        commit?: boolean, // That is the additional filter to indicate the template needs the commit.
        // Even though we don't have it in our documentation.
        repo?: string,
        repo_branch?: string | { repo: string, branch: string }[],
        branch?: string,
        branches?: Branch[],
        search?: string,
        fields?: string[],
        searchFields?: string[],
        limit?: number,
        offset?: number,
        sort?: string,
        type?: string[],
        dedup?: boolean,
    }, showLoader =  true) {
        let customHeader = new Headers();
        let search = new URLSearchParams();
        params = params || {};

        search.set('sort', 'name');

        // Loader is shown by default
        if (!showLoader) {
            customHeader.append('isUpdated', 'true');
        }

        if (params.commit) {
            search.set('commit', '');
        }

        if (params.repo) {
            search.set('repo', params.repo.toString());
        }

        if (params.branch) {
            search.set('branch', params.branch.toString());
        }

        if (params.branches) {
            search.set('repo_branch', JSON.stringify(params.branches.map(item => {
                return { repo: item.repo, branch: item.name };
            })));
        }

        if (params.repo_branch) {
            if (typeof params.repo_branch === 'string') {
                search.set('repo_branch', params.repo_branch);
            } else {
                search.set('repo_branch', JSON.stringify(params.repo_branch));
            }
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

        if (params.fields) {
            search.set('fields', params.fields.join(','));
        }

        if (params.searchFields) {
            search.set('search_fields', params.searchFields.join(','));
        }

        if (params.search) {
            search.set('search', '~' + params.search.toString());
        }

        if (params.type) {
            search.set('type', params.type.join(','));
        }

        if (params.dedup) {
            search.set('dedup', 'true');
        }

        return this._http.get(`v1/templates`, { search: search, headers: customHeader }).map(res => res.json());
    }

    getTemplateByIdAsync(templateId, noErrorHandling = false): Observable<Template> {
        return this._http.get(`v1/templates/${templateId}`, { headers: new AxHeaders({ noErrorHandling }) }).map(res => res.json());
    }
}
