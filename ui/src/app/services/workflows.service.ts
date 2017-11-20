import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';

import * as models from '../models';
import { WorkflowList } from '../models';

@Injectable()
export class WorkflowsService {

  constructor(private http: HttpClient) { }

  public async getWorkflows(): Promise<models.WorkflowList> {
      return this.http.get('apis/argoproj.io/v1/workflows').map(item => <WorkflowList>item).toPromise();
  }
}
