import { Injectable, Optional, NgZone } from '@angular/core';
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
  constructor(private http: HttpClient, private zone: NgZone) { }

    /**
     * Reads server sent messages from specified URL.
     */
    private loadEventSource(url: string, zone?: NgZone): Observable<string> {
      function runInZone(action: () => any) {
        if (zone) {
          zone.run(() => action());
        } else {
          action();
        }
      }
      return Observable.create((observer: Observer<any>) => {
          const eventSource = new EventSource(url);
          eventSource.onmessage = msg => runInZone(() => observer.next(msg.data));
          eventSource.onerror = e => runInZone(() => {
              if (e.eventPhase === ReadyState.CLOSED || eventSource.readyState === ReadyState.CONNECTING) {
                  observer.complete();
              } else {
                  observer.error(e);
              }
          });
          return () => {
              runInZone(() => eventSource.close());
          };
      });
  }

  public async getWorkflows(statuses: string[] = []): Promise<models.WorkflowList> {
      return this.http.get(`api/workflows`, { params: { status: statuses } }).map(item => <WorkflowList>item).toPromise();
  }

  public async getWorkflow(namespace: string, name: string, noLoader = false): Promise<models.Workflow> {
    return this.http.get(
      `api/workflows/${namespace}/${name}`,
      { headers: new HttpHeaders({ noLoader: String(noLoader) }) }).map(item => <Workflow>item).toPromise();
  }

  public getWorkflowStream(namespace: string, name: string): Observable<models.Workflow> {
    return Observable.merge(
      Observable.fromPromise(this.getWorkflow(namespace, name, false)),
      this.loadEventSource(
        `api/workflows/live?name=${name}&namespace=${namespace}`, this.zone).repeat().retry().map(data => JSON.parse(data)));
  }

  public getWorkflowsStream(): Observable<models.Workflow> {
    return this.loadEventSource('api/workflows/live', this.zone).repeat().retry().map(data => JSON.parse(data));
  }

  public getStepLogs(namespace: string, name: string): Observable<string> {
    return this.loadEventSource(`api/steps/${namespace}/${name}/logs`).map(line => {
      return line ? line + '\n' : line;
    });
  }

  public connectToConsole(uri: string, params: URLSearchParams) {
    const search = params || new URLSearchParams();
    const scheme = location.protocol === 'http:' ? 'ws' : 'wss';
    const socket = new WebSocket(`${scheme}://${location.hostname}:${location.port}/${uri}?${search.toString()}`);
    socket.binaryType = 'arraybuffer';
    return socket;
  }
}
