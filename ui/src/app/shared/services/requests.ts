import * as _superagent from 'superagent';

const superagentPromise = require('superagent-promise');
import {Observable, Observer} from 'rxjs';

import {apiUrl} from '../base';

type Callback = (data: any) => void;

declare class EventSource {
    public onopen: Callback;
    public onmessage: Callback;
    public onerror: Callback;
    public readyState: number;
    constructor(url: string);
    public close(): void;
}

enum ReadyState {
    CONNECTING = 0,
    OPEN = 1,
    CLOSED = 2,
    DONE = 4
}

function token() {
    if (localStorage.getItem('token') === null) {
        localStorage.setItem('token', window.prompt('Please copy and paste your ~/.kube/config base 64 encoded.' + 'cat ${KUBECONFIG:-~/.kube/config} | base64 | pbcopy'));
    }
    return localStorage.getItem('token');
}

const superagent: _superagent.SuperAgentStatic = superagentPromise(_superagent, global.Promise);

export default {
    get(url: string) {
        return superagent.get(apiUrl(url)).auth(token(), {type: 'bearer'});
    },

    post(url: string) {
        return superagent.post(apiUrl(url)).auth(token(), {type: 'bearer'});
    },

    put(url: string) {
        return superagent.put(apiUrl(url)).auth(token(), {type: 'bearer'});
    },

    patch(url: string) {
        return superagent.patch(apiUrl(url)).auth(token(), {type: 'bearer'});
    },

    loadEventSource(url: string, allowAutoRetry = false): Observable<string> {
        return Observable.create((observer: Observer<any>) => {
            const eventSource = new EventSource(apiUrl(url));
            let opened = false;
            eventSource.onopen = msg => {
                if (!opened) {
                    opened = true;
                } else if (!allowAutoRetry) {
                    eventSource.close();
                    observer.complete();
                }
            };
            eventSource.onmessage = msg => observer.next(msg.data);
            eventSource.onerror = e => () => {
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
};
