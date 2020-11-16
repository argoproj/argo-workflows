import * as kubernetes from 'argo-ui/src/models/kubernetes';
import {Observable} from 'rxjs';
import {RetryWatch} from './retry-watch';

interface Resource {
    metadata: kubernetes.ObjectMeta;
}

type Type = 'ADDED' | 'MODIFIED' | 'DELETED' | 'ERROR';
type Sorter = (a: Resource, b: Resource) => number;

// alphabetical name order
export const sortByName: Sorter = (a: Resource, b: Resource) => (a.metadata.name > b.metadata.name ? -1 : 1);
// put the youngest at the start of the list
export const sortByYouth: Sorter = (a: Resource, b: Resource) =>
    a.metadata.creationTimestamp === b.metadata.creationTimestamp ? 0 : a.metadata.creationTimestamp < b.metadata.creationTimestamp ? -1 : 1;

const reconnectAfterMs = 3000;

/**
 * ListWatch allows you to start watching for changes, automatically reconnecting on error.
 *
 * Items are sorted by creation timestamp.
 */
export class ListWatch<T extends Resource> {
    private readonly list: () => Promise<{metadata: kubernetes.ListMeta; items: T[]}>;
    private readonly onLoad: (metadata: kubernetes.ListMeta) => void;
    private readonly onChange: (items: T[]) => void;
    private readonly onError: (error: Error) => void;
    private readonly sorter: (a: T, b: T) => number;
    private items: T[];
    private retryWatch: RetryWatch<T>;
    private timeout: any;

    constructor(
        list: () => Promise<{metadata: kubernetes.ListMeta; items: T[]}>,
        watch: (resourceVersion: string) => Observable<kubernetes.WatchEvent<T>>,
        onLoad: (metadata: kubernetes.ListMeta) => void, // called when the list is loaded
        onOpen: () => void, //  called, when watches is re-established after error,  so should clear any errors
        onChange: (items: T[]) => void, // called whenever items change
        onError: (error: Error) => void, // called on any error
        sorter: Sorter = sortByName
    ) {
        this.onLoad = onLoad;
        this.list = list;
        this.retryWatch = new RetryWatch<T>(
            watch,
            onOpen,
            e => {
                this.items = mergeItem(e.object, e.type, this.items).sort(this.sorter);
                onChange(this.items);
            },
            onError
        );
        this.onChange = onChange;
        this.onError = onError;
        this.sorter = sorter;
    }

    // Start watching
    // Idempotent.
    public start() {
        this.list()
            .then(x => {
                this.items = (x.items || []).sort(this.sorter);
                this.onLoad(x.metadata);
                this.onChange(this.items);
                this.retryWatch.start(x.metadata.resourceVersion);
            })
            .catch(e => {
                clearTimeout(this.timeout);
                this.onError(e);
                this.timeout = setTimeout(() => this.start(), reconnectAfterMs);
            });
    }

    // Stop watching.
    // You should almost always  invoke on component unload.
    // Idempotent.
    public stop() {
        this.retryWatch.stop();
    }
}

/**
 * This is used to update (or delete) and item in a the list.
 */
const mergeItem = <T extends Resource>(item: T, type: Type, items: T[]): T[] => {
    const index = items.findIndex(x => x.metadata.uid === item.metadata.uid);
    if (type === 'DELETED') {
        if (index > -1) {
            items.splice(index, 1);
        }
    } else if (type !== 'ERROR') {
        if (index > -1) {
            items[index] = item;
        } else {
            items.unshift(item);
        }
    }
    return items;
};
