import * as kubernetes from 'argo-ui/src/models/kubernetes';
import {Observable, Subscription} from 'rxjs';

interface Resource {
    metadata: kubernetes.ObjectMeta;
}

const reconnectAfterMs = 5000;

/**
 * RetryWatch allows you to watch for changes, automatically reconnecting on error.
 */
export class RetryWatch<T extends Resource> {
    private readonly watch: (resourceVersion: string) => Observable<kubernetes.WatchEvent<T>>;
    private readonly onOpen: () => void;
    private readonly onItem: (event: kubernetes.WatchEvent<T>) => void;
    private readonly onError: (error: Error) => void;
    private subscription: Subscription;
    private timeout: any; // should be `number`

    constructor(
        watch: (resourceVersion: string) => Observable<kubernetes.WatchEvent<T>>,
        onOpen: () => void, //  called when watches (re-)established after error, so should clear any errors
        onEvent: (event: kubernetes.WatchEvent<T>) => void, // called whenever item is received,
        onError: (error: Error) => void
    ) {
        this.watch = watch;
        this.onOpen = onOpen;
        this.onItem = onEvent;
        this.onError = onError;
    }

    public start(resourceVersion: string) {
        this.stop();
        this.subscription = this.watch(resourceVersion).subscribe(
            next => {
                if (next) {
                    this.onItem(next);
                } else {
                    this.onOpen();
                }
            },
            e => {
                clearTimeout(this.timeout);
                this.onError(e);
                this.timeout = setTimeout(() => this.start('0'), reconnectAfterMs);
            }
        );
    }

    // Must invoke on component unload.
    public stop() {
        clearTimeout(this.timeout);
        if (this.subscription) {
            this.subscription.unsubscribe();
        }
    }
}
