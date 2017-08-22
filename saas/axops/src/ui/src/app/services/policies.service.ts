import { Inject, Injectable } from '@angular/core';
import { Http, URLSearchParams, Headers } from '@angular/http';

@Injectable()
export class PoliciesService {
    constructor(@Inject(Http) private _http) {
    }

    getPolicies(params?: {
        name?: string,
        description?: string,
        repo?: string,
        branch?: string,
        template?: string,
        enabled?: boolean,
        deleted?: boolean,
        search?: string,
        limit?: number,
        offset?: number,
        repo_branch?: string | {repo: string, branch: string}[],
        status?: string,
    }, hideLoader?: boolean) {
        let customHeader = new Headers();
        let filter = new URLSearchParams();
        if (hideLoader) {
            customHeader.append('isUpdated', hideLoader.toString());
        }
        if (params.name) {
            filter.set('name', params.name.toString());
        }
        if (params.description) {
            filter.set('description', params.description.toString());
        }
        if (params.repo) {
            filter.set('repo', params.repo.toString());
        }
        if (params.branch) {
            filter.set('branch', params.branch.toString());
        }
        if (params.template) {
            filter.set('template', params.template.toString());
        }
        if (params.enabled !== undefined) {
            filter.set('enabled', params.enabled.toString());
        }
        if (params.deleted !== undefined) {
            filter.set('deleted', params.deleted.toString());
        }
        if (params.search) {
            filter.set('search', `~${params.search.toString()}`);
        }
        if (params.limit) {
            filter.set('limit', params.limit.toString());
        }
        if (params.offset) {
            filter.set('offset', params.offset.toString());
        }
        if (params.repo_branch) {
            if (typeof params.repo_branch === 'string') {
                filter.set('repo_branch', params.repo_branch);
            } else {
                filter.set('repo_branch', JSON.stringify(params.repo_branch));
            }
        }
        if (params.status) {
            filter.set('status', params.status.toString());
        }

        return this._http.get(`v1/policies?${filter.toString()}`, {headers: customHeader})
            .map(res => res.json());
    }

    getPolicyById(commitPolicyId) {
        return this._http.get(`v1/policies/${commitPolicyId}`)
            .map(res => res.json());
    }

    enablePolicy(policyId) {
        return this._http.put(`v1/policies/${policyId}/enable`)
            .map(res => res.json());
    }

    disablePolicy(policyId) {
        return this._http.put(`v1/policies/${policyId}/disable`)
            .map(res => res.json());
    }

    deletePolicy(policyId) {
        return this._http.delete(`v1/policies/${policyId}`);
    }
}
