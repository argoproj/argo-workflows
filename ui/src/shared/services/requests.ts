import {Observable, Observer} from 'rxjs';
import * as superagent from 'superagent';
import {SuperAgentRequest} from 'superagent';

import {apiUrl, uiUrlWithParams} from '../base';

// Add a timeout to all requests to prevent hanging
const REQUEST_TIMEOUT = 20000; // 20 seconds

// Track if we're currently in a page reload to avoid multiple redirects
let isReloading = false;

function auth(req: SuperAgentRequest) {
    return req.timeout(REQUEST_TIMEOUT).on('error', handle);
}

function handle(err: any) {
    console.error('API request error:', err);

    // Prevent multiple redirects or handling during page reload
    if (isReloading) {
        return;
    }

    // check URL to prevent redirect loop
    if (err.status === 401 && !document.location.href.includes('login')) {
        isReloading = true;
        document.location.href = uiUrlWithParams('login', ['redirect=' + document.location.href]);
        return;
    }

    // Handle timeout errors specifically
    if (err.timeout) {
        console.warn('Request timed out. This might cause UI issues.');
    }
}

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

    loadEventSource(url: string): Observable<string> {
        return new Observable((observer: Observer<any>) => {
            const eventSource = new EventSource(url);
            // an null event is the best way I could find to get an event whenever we open the event source
            // otherwise, you'd have to wait for your first message (which maybe some time)
            eventSource.onopen = () => observer.next(null);
            eventSource.onmessage = x => observer.next(x.data);
            eventSource.onerror = () => {
                switch (eventSource.readyState) {
                    case EventSource.CONNECTING:
                        observer.error(new Error('Failed to connect to ' + url));
                        break;
                    case EventSource.OPEN:
                        observer.error(new Error('Error in open connection to ' + url));
                        break;
                    case EventSource.CLOSED:
                        observer.error(new Error('Connection closed to ' + url));
                        break;
                    default:
                        observer.error(new Error('Unknown error with ' + url));
                }
            };

            return () => {
                eventSource.close();
            };
        });
    }
};
