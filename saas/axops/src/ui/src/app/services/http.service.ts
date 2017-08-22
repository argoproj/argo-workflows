import {Observable, Observer} from 'rxjs';
import {Injectable, NgZone} from '@angular/core';

let zlib = require('zlib');
let untar = require('js-untar/build/dist/untar.js');

interface Callback {(data: any): void; }

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

/**
 * Implements specific low level http requests e.g. reading blob or server sent events.
 */
export class HttpService {
    constructor(private zone: NgZone) {}

    /**
     * Loads and unpack tarball.
     */
    loadTar(url): Observable<{name: string, blob: Blob}> {
        return Observable.create((observer: Observer<{name: string, blob: Blob}>) => {
            let zone = this.zone;

            let xhr = new XMLHttpRequest();
            xhr.onreadystatechange = function() {
                if (this.readyState === ReadyState.DONE && this.status === 200) {
                    zlib.gunzip(new Buffer(this.response), (error, data) => {
                        untar(data.buffer).then(extractedFiles => {
                            zone.run(() => {
                                extractedFiles.forEach(file => observer.next(file));
                                observer.complete();
                            });
                        }).catch(err => {
                            zone.run(() => observer.error(err));
                        });
                    });
                }
            };
            xhr.open('GET', url);
            xhr.responseType = 'arraybuffer';
            xhr.send();

            return () => { xhr.abort(); };
        });
    }

    /**
     * Reads server sent messages from specified URL.
     */
    loadEventSource(url): Observable<string> {
        return Observable.create((observer: Observer<any>) => {
            let eventSource = new EventSource(url);
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
}
