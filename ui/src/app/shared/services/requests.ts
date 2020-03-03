// @ts-ignore
import {EventSourcePolyfill} from 'event-source-polyfill';
import {Observable, Observer} from 'rxjs';
import * as _superagent from 'superagent';
import {SuperAgentRequest} from 'superagent';
import {apiUrl, uiUrl} from '../base';

const superagentPromise = require('superagent-promise');

const auth = (req: SuperAgentRequest) => {
    return req.on('error', handle);
};

const handle = (err: any) => {
    if (err.status === 401) {
        document.location.href = uiUrl('login');
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

    delete(url: string) {
        return auth(superagent.del(apiUrl(url)));
    },

    loadEventSource(url: string, allowAutoRetry = false): Observable<string> {
        return Observable.create((observer: Observer<any>) => {
            const eventSource = new EventSource(url);
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
                if (e.eventPhase === Event.AT_TARGET) {
                    if (!allowAutoRetry) {
                        observer.complete();
                    }
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
