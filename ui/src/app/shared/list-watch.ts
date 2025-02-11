import * as kubernetes from 'argo-ui/src/models/kubernetes';
import {Observable} from 'rxjs';
import {RetryWatch} from './retry-watch';

interface Resource {
    metadata: kubernetes.ObjectMeta;
}

type Type = 'ADDED' | 'MODIFIED' | 'DELETED' | 'ERROR';
type Sorter = (a: Resource, b: Resource) => number;

// put the youngest at the start of the list
export const sortByYouth: Sorter = (a: Resource, b: Resource) => b.metadata.creationTimestamp.localeCompare(a.metadata.creationTimestamp);

/**
 * ListWatch allows you to start watching for changes, automatically reconnecting on error.
 */
export class ListWatch<T extends Resource> {
    private readonly list: () => Promise<{metadata: kubernetes.ListMeta; items: T[]}>;
    private readonly onLoad: (metadata: kubernetes.ListMeta) => void;
    private readonly onChange: (items: T[], item?: T, type?: Type) => void;
    private readonly onError: (error: Error) => void;
    private readonly sorter: (a: T, b: T) => number;
    private items: T[];
    private retryWatch: RetryWatch<T>;
    private timeout: any;
    private reconnectAfterMs = 3000;

    constructor(
        list: () => Promise<{metadata: kubernetes.ListMeta; items: T[]}>,
        watch: (resourceVersion?: string) => Observable<kubernetes.WatchEvent<T>>,
        onLoad: (metadata: kubernetes.ListMeta) => void, // called when the list is loaded
        onOpen: () => void, //  called, when watches is re-established after error,  so should clear any errors
        onChange: (items: T[], item?: T, type?: Type) => void, // called whenever items change, any users that changes state should use [...items]
        onError: (error: Error) => void, // called on any error
        sorter: Sorter = sortByYouth // show the youngest first by default
    ) {
        this.onLoad = onLoad;
        this.list = list;
        this.onChange = onChange;
        this.onError = onError;
        this.sorter = sorter;
        this.retryWatch = new RetryWatch<T>(
            watch,
            onOpen,
            e => {
                this.items = mergeItem(e.object, e.type, this.items).sort(sorter);
                onChange(this.items, e.object, e.type);
            },
            onError
        );
    }

    // Start watching
    // Idempotent.
    public start() {
        this.stop();
        this.list()
            .then(x => {
                this.items = (x.items || []).sort(this.sorter);
                this.onLoad(x.metadata);
                this.onChange(this.items);
                this.retryWatch.start(x.metadata.resourceVersion);
            })
            .catch(e => {
                this.stop();
                this.onError(e);
                this.reconnect();
            });
    }

    // Stop watching.
    // Must invoke on component unload.
    // Idempotent.
    public stop() {
        clearTimeout(this.timeout);
        this.retryWatch.stop();
    }

    private reconnect() {
        this.timeout = setTimeout(() => this.start(), this.reconnectAfterMs);
        this.reconnectAfterMs = Math.min(this.reconnectAfterMs * 1.5, 60000);
    }
}

/**
 * This is used to update (or delete) and item in a the list.
 */
const mergeItem = <T extends Resource>(item: T, type: Type, items: T[]): T[] => {
    const index = items.findIndex(x => x.metadata.namespace === item.metadata.namespace && x.metadata.name === item.metadata.name);
    if (type === 'DELETED') {
        if (index > -1) {
            items.splice(index, 1);
        }
    } else if (type !== 'ERROR') {
        if (index > -1) {
            items[index] = item;
        } else {
            items.push(item);
        }
    }
    return items;
};
