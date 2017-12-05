import { Injectable, Optional } from '@angular/core';
import { HttpClient } from '@angular/common/http';

import * as models from '../models';
import { WorkflowList, Workflow } from '../models';
import { Observable } from 'rxjs/Observable';
import { timeInterval } from 'rxjs/operators';
import { Observer } from 'rxjs/Observer';
import { HttpHeaders } from '@angular/common/http';

type Callback = (data: any) => void;

declare class EventSource {
    onmessage: Callback;
    onerror: Callback;
    readyState: number;
    close(): void;
    constructor(url: string);
}

enum ReadyState {
  CONNECTING = 0,
  OPEN = 1,
  CLOSED = 2,
  DONE = 4
}

@Injectable()
export class WorkflowsService {
  constructor(private http: HttpClient) { }

    /**
     * Reads server sent messages from specified URL.
     */
    private loadEventSource(url): Observable<string> {
      return Observable.create((observer: Observer<any>) => {
          const eventSource = new EventSource(url);
          eventSource.onmessage = msg => observer.next(msg.data);
          eventSource.onerror = e => {
              if (e.eventPhase === ReadyState.CLOSED || eventSource.readyState === ReadyState.CONNECTING) {
                  observer.complete();
              } else {
                  observer.error(e);
              }
          };
          return () => {
              eventSource.close();
          };
      });
  }

  public async getWorkflows(): Promise<models.WorkflowList> {
      return this.http.get(`api/workflows`).map(item => <WorkflowList>item).toPromise();
  }

  public async getWorkflow(namespace: string, name: string, noLoader = false): Promise<models.Workflow> {
    return this.http.get(
      `api/workflows/${namespace}/${name}`,
      { headers: new HttpHeaders({ noLoader: String(noLoader) }) }).map(item => <Workflow>item).toPromise();
  }

  public getWorkflowStream(namespace: string, name: string): Observable<models.Workflow> {
    return Observable.merge(
      Observable.fromPromise(this.getWorkflow(namespace, name, false)),
      Observable.interval(1000).flatMap(() => Observable.fromPromise(this.getWorkflow(namespace, name, true))).distinct(workflow => {
        return Object.keys(workflow.status.nodes || []).map(nodeName => `${nodeName}:${workflow.status.nodes[nodeName].phase}`).join(';');
      }));
  }

  public getStepLogs(namespace: string, name: string): Observable<string> {
    return this.loadEventSource(`api/steps/${namespace}/${name}/logs`).map(line => {
      return line ? line + '\n' : line;
    });
  }
}
