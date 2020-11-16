import * as kubernetes from 'argo-ui/src/models/kubernetes';
import {Observable, Subscription} from 'rxjs';

type Object = { metadata: kubernetes.ObjectMeta }
type Type = 'ADDED' | 'MODIFIED' | 'DELETED' | 'ERROR'
type Sorter = (a: Object, b: Object) => number


// alphabetical name order
export const sortByName: Sorter = (a: Object, b: Object) => a.metadata.name > b.metadata.name ? -1 : 1
// put the youngest at the start of the list
export const sortByYouth: Sorter = (a: Object, b: Object) => a.metadata.creationTimestamp === b.metadata.creationTimestamp ? 0 : a.metadata.creationTimestamp < b.metadata.creationTimestamp ? -1 : 1

const reconnectAfterMs = 3000;

/**
 * ListWatch allows you to start watching for changes, automatically reconnecting on error.
 *
 * Items are sorted by creation timestamp.
 */
export class ListWatch<T extends Object> {
    private readonly list: () => Promise<{ metadata: kubernetes.ListMeta; items: T[] }>;
    private readonly watch: (resourceVersion: string) => Observable<{ object: T; type: Type }>;
    private readonly onLoad: (metadata: kubernetes.ListMeta) => void;
    private readonly onChange: (items: T[]) => void;
    private readonly onError: (error: Error) => void;
    private readonly sorter: (a: T, b: T) => number;
    private items: T[];
    private lastResourceVersion: string;
    private subscription: Subscription;
    private timeout: any; // should be `number`

    constructor(
        list: () => Promise<{ metadata: kubernetes.ListMeta, items: T[] }>,
        watch: (resourceVersion: string) => Observable<{ object: T, type: Type }>,
        onLoad: (metadata: kubernetes.ListMeta) => void, // called when the list is loaded
        onChange: (items: T[]) => void, // called whenever items change, should clear any errors
        onError: (error: Error) => void, // called on any error
        sorter: Sorter = sortByName
    ) {
        this.onLoad = onLoad;
        this.list = list;
        this.watch = watch;
        this.onChange = onChange;
        this.onError = onError;
        this.sorter = sorter
    }

    // Start watching
    // Idempotent.
    start() {
        this.list()
            .then(x => {
                console.log('load')
                this.items = x.items.sort(this.sorter);
                this.lastResourceVersion = x.metadata.resourceVersion;
                this.onLoad(x.metadata)
                this.onChange(this.items)
                this.startWatching()
            })
            .catch(e => {
                console.log('list error', e)
                clearTimeout(this.timeout);
                this.onError(e)
                this.timeout = setTimeout(() => this.start(), reconnectAfterMs)
            })
    }

    private startWatching() {
        this.stopWatching();
        this.subscription = this.watch(this.lastResourceVersion, () => this.onError(null))
            .subscribe(next => {
                    console.log('next', next)
                    this.items = mergeItem(next.object, next.type, this.items).sort(this.sorter);
                    this.lastResourceVersion = next.object.metadata.resourceVersion;

                    this.onChange(this.items)
                },
                e => {
                    console.log('watch error', e)
                    clearTimeout(this.timeout);
                    this.onError(e)
                    this.timeout = setTimeout(() => this.startWatching(), reconnectAfterMs)
                }
            )
    }

    // Stop watching.
    // You should almost always  invoke on component unload.
    // Idempotent.
    stop() {
        this.stopWatching();
    }

    private stopWatching() {
        if (this.subscription) {
            this.subscription.unsubscribe()
        }
    }
}


/**
 This is used to update (or delete) and item in a the list.
 */
const mergeItem = <T extends Object>(item: T, type: Type, items: T[]): T[] => {
    const index = items.findIndex(item => item.metadata.uid === item.metadata.uid);
    if (type === 'DELETED') {
        if (index > -1) {
            items.splice(index, 1);
        }
    } else if (type !== 'ERROR') {
        if (index > -1) {
            items[index] = item
        } else {
            items.unshift(item);
        }
    }
    return items;
}