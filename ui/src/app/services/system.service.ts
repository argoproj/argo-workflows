import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';

import * as models from '../models';

@Injectable()
export class SystemService {

  constructor(private http: HttpClient) { }

  public getSettings(): Promise<models.Settings> {
    return this.http.get('api/system/settings').map(item => <models.Settings>item).toPromise();
  }
}
