import {Component, Input} from '@angular/core';
import {DomSanitizer, SafeResourceUrl} from '@angular/platform-browser';
import {HttpService} from '../../services';

const SCOPE = '/assets/workers/';
const WORKER_URL = '/assets/workers/zip-viewer-worker.js';
let serviceWorkerInitPromise: Promise<any> = null;

declare class Response {
    constructor(data: any);
}

@Component({
    selector: 'ax-zip-viewer',
    templateUrl: './zip-viewer.html'
})
/**
 * Downloads resource using specified URL and render it in embedded iframe. URL should point to tar file.
 * Component unpack it and place each file into window cache. Populated cache is used by zip-viewer-worker.js to render tar content in
 * iframe.
 */
export class ZipViewerComponent {

    indexCacheUrl: SafeResourceUrl;
    downloadUrl: string;
    state: 'loading' | 'error' | 'ready' = 'loading';

    constructor(private httpService: HttpService, private domSanitizationService: DomSanitizer) {
    }

    @Input()
    set url(value: string) {
        if (value && value !== this.downloadUrl) {
            this.downloadUrl = value;
            this.refresh();
        }
    }

    private refresh() {
        this.state = 'loading';
        this.indexCacheUrl = null;
        let cachedUrls: string[] = [];
        this.ensureWorkerInitialized().then(caches => caches.open('ax-zip-viewer-v1')).then(cache => {
            this.httpService.loadTar(this.downloadUrl).subscribe(file => {
                let cacheUrl = `${SCOPE}zip-viewer/${file.name}`;
                cachedUrls.push(cacheUrl);
                cache.put(cacheUrl, new Response(file.blob));
            }, () => {
                this.state = 'error';
            }, () => {
                let url = cachedUrls.find(item => item.indexOf('index.html') > -1);
                if (url) {
                    this.indexCacheUrl = this.domSanitizationService.bypassSecurityTrustResourceUrl(url);
                }
                this.state = 'ready';
            });
        }).catch(() => {
            this.state = 'error';
        });
    }

    private ensureWorkerInitialized(): Promise<any> {
        if (!serviceWorkerInitPromise) {
            let serviceWorker = navigator['serviceWorker'];
            let registerWorkerPromise = serviceWorker ?
                <Promise<any>>serviceWorker.register(WORKER_URL, { scope: SCOPE }) :
                Promise.reject('Service worker is not supported');
            serviceWorkerInitPromise = registerWorkerPromise.then(() => {
                let caches = window['caches'];
                if (!caches) {
                    throw 'Window caches in not supported';
                }
                return caches;
            });
        }
        return serviceWorkerInitPromise;
    }
}
