import * as _superagent from 'superagent';

const superagentPromise = require('superagent-promise');
import {Observable, Observer} from 'rxjs';

import {SuperAgentRequest} from 'superagent';
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

const auth = (req: SuperAgentRequest) => {
    const token = localStorage.getItem('token');
    return (token !== null ? req.auth(token, {type: 'bearer'}) : req).on('error', handle);
};

const handle = (err: any) => {
    if (err.status === 401) {
        document.location.href = '/login';
    }
};

const superagent: _superagent.SuperAgentStatic = superagentPromise(_superagent, global.Promise);

export default {
    get(url: string) {
        return auth(superagent.get(apiUrl(url)));
    },

    post(url: string) {
        return auth(superagent.post(apiUrl(url)));
    },

    put(url: string) {
        return auth(superagent.put(apiUrl(url)));
    },

    patch(url: string) {
        return auth(superagent.patch(apiUrl(url)));
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
