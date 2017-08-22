import { Inject, Injectable } from '@angular/core';
import { Http, Headers, URLSearchParams } from '@angular/http';
import { Observable } from 'rxjs/Observable';
import { Branch } from '../model';

@Injectable()
export class BranchService {
    constructor(@Inject(Http) private http) {
    }

    getBranchesAsync(params: {
        limit?: number,
        name?: string,
        repo?: string,
        branch?: string,
        orderBy?: string
    }, hideLoader = false): Observable<{data: Branch[]}> {
        let customHeader = new Headers();
        let filter = new URLSearchParams();

        if (hideLoader) {
            customHeader.append('isUpdated', 'true');
        }

        if (params.limit) {
            filter.set('limit', params.limit.toString());
        }
        if (params.name) {
            filter.set('name', params.name.toString());
        }
        if (params.repo) {
            filter.set('repo', params.repo.toString());
        }
        if (params.branch) {
            filter.set('branch', params.branch.toString());
        }
        if (params.orderBy) {
            filter.set('order_by', params.orderBy.toString());
        }

        return this.http.get('v1/branches', {headers: customHeader, search: filter})
            .map(res => res.json());
    }

    getBranchByRepoIdAsync(repoId) {
        return this.http.get(`v1/branches?repo=${repoId}&session`)
            .map(res => res.json());
    }

    updateBranchAsync(branchId, branch) {
        return this.http.put(`v1/branches/${branchId}`, JSON.stringify(branch))
            .map(res => res.json());
    }
}
