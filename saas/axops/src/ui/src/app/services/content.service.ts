import * as moment from 'moment';
import { Injectable } from '@angular/core';
import { Http, Headers, URLSearchParams } from '@angular/http';

export interface DocUrls {
    releaseNotesUrl: string;
    quickStartUrl: string;
    yamlDslUrl: string;
}

@Injectable()
export class ContentService {

    constructor(private http: Http, private baseUrlPromise: Promise<string>) {}

    public getTutorial(tutorialName: string): Promise<string> {
        return this.loadContent(`tutorials/${tutorialName}.md`);
    }

    public getDocUrls(): Promise<DocUrls> {
        return this.baseUrlPromise.then(baseUrl => ({
            releaseNotesUrl: `${baseUrl}/release-notes.pdf`,
            quickStartUrl: `${baseUrl}/quick-start.pdf`,
            yamlDslUrl: `${baseUrl}/yaml-dsl.pdf`,
            userGuideUrl: `${baseUrl}/user-guide.pdf`
        }));
    }

    private loadContent(path: string): Promise<string> {
        let search = new URLSearchParams();
        // Bust cache every hour
        search.set('_', moment().endOf('hour').unix().toString());
        return this.baseUrlPromise.then(baseUrl => this.http.get(`${baseUrl}/${path}`, {
            headers: new Headers({isUpdated: true}),
            search
        }).toPromise().then(res => res.text()));
    }
}
