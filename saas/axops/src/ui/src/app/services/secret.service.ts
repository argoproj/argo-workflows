import { Injectable } from '@angular/core';
import { Http, Headers } from '@angular/http';

import { AuthenticationService } from './authentication.service';

@Injectable()
export class SecretService {
    constructor(private http: Http, private authenticationService: AuthenticationService) {}

    public async encrypt(secret: string, repo: string): Promise<string> {
        let headers = new Headers();
        headers.append('isUpdated', 'true');
        let res = await this.http.post('v1/secret/encrypt', { plain_text: { plaintext: secret, repo } }, { headers }).toPromise();
        return res.json().cipher_text.encrypted_content;
    }

    public getKey(): Promise<string> {
        let session = this.authenticationService.getSessionToken();
        return this.http.post('v1/secret/key', { session }).toPromise().then(res => res.json().key);
    }

    public updateKey(key: string): Promise<any> {
        return this.http.put('v1/secret/key', { key }).toPromise();
    }
}
