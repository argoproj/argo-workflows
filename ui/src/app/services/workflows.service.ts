import { Injectable, Optional } from '@angular/core';
import { HttpClient } from '@angular/common/http';

import * as models from '../models';
import { WorkflowList, Workflow } from '../models';
import { Observable } from 'rxjs/Observable';
import { timeInterval } from 'rxjs/operators';

@Injectable()
export class WorkflowsService {

  private namespace = 'default';

  constructor(private http: HttpClient) { }

  public async getWorkflows(): Promise<models.WorkflowList> {
      return this.http.get(`apis/argoproj.io/v1/namespaces/${this.namespace}/workflows`).map(item => <WorkflowList>item).toPromise();
  }

  public async getWorkflow(name: string): Promise<models.Workflow> {
    return this.http.get(`apis/argoproj.io/v1/namespaces/${this.namespace}/workflows/${name}`).map(item => <Workflow>item).toPromise();
  }

  public getWorkflowStream(name: string): Observable<models.Workflow> {
    return Observable.interval(1000).flatMap(() => Observable.fromPromise(this.getWorkflow(name)));
  }
}
