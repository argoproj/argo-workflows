import {Inject, Injectable} from '@angular/core';
import {Http, Headers} from '@angular/http';

@Injectable()
export class RepoService {
    constructor(@Inject(Http) private _http) {
    }

    getReposAsync(hideLoader = false ) {
        let customHeader = new Headers();
        if (hideLoader) {
            customHeader.append('isUpdated', 'true');
        }
        return this._http.get('v1/repos', {headers: customHeader})
            .map(res => res.json());
    }
}
