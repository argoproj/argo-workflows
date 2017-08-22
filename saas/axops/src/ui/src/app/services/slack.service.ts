import { Injectable } from '@angular/core';
import { Http } from '@angular/http';

import { AxHeaders } from './headers';

@Injectable()
export class SlackService {
    constructor(private http: Http) {
    }

    public async getSlackChannels(): Promise<string[]> {
        return this.http.get(`v1/slack/channels`, {headers: new AxHeaders({noLoader: true})}).toPromise().then(res => res.json().data);
    }
}
