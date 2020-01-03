// @ts-ignore
import {EventSourcePolyfill} from 'event-source-polyfill';
import {Observable, Observer} from 'rxjs';
import * as _superagent from 'superagent';
import {SuperAgentRequest} from 'superagent';

const superagentPromise = require('superagent-promise');

enum ReadyState {
    CONNECTING = 0,
    OPEN = 1,
    CLOSED = 2,
    DONE = 4
}

const getToken = () => localStorage.getItem('token');

const auth = (req: SuperAgentRequest) => {
    const token = getToken();
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
        return auth(superagent.get(url));
    },

    post(url: string) {
        return auth(superagent.post(url));
    },

    put(url: string) {
        return auth(superagent.put(url));
    },

    patch(url: string) {
        return auth(superagent.patch(url));
    },

    delete(url: string) {
        return auth(superagent.del(url));
    },

    loadEventSource(url: string, allowAutoRetry = false): Observable<string> {
        return Observable.create((observer: Observer<any>) => {
            const token = getToken();
            const headers: any = {};
            if (token !== null) {
                headers.Authorization = `Bearer ${getToken()}`;
            }
            const eventSource = new EventSourcePolyfill(url, {headers});
            let opened = false;
            eventSource.onopen = (msg: any) => {
                if (!opened) {
                    opened = true;
                } else if (!allowAutoRetry) {
                    eventSource.close();
                    observer.complete();
                }
            };
            eventSource.onmessage = (msg: any) => observer.next(msg.data);
            eventSource.onerror = (e: any) => {
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
