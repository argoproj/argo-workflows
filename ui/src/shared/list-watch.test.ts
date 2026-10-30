import {EMPTY} from 'rxjs';

import {ListWatch} from './list-watch';

function deferred<T>() {
    let resolve: (value: T) => void;
    let reject: (reason?: unknown) => void;
    const promise = new Promise<T>((res, rej) => {
        resolve = res;
        reject = rej;
    });
    return {promise, resolve: resolve!, reject: reject!};
}

describe('ListWatch', () => {
    afterEach(() => {
        jest.useRealTimers();
    });

    it('does not publish or watch a list result after it has been stopped', async () => {
        const request = deferred<{metadata: {resourceVersion: string}; items: Array<{metadata: {name: string; namespace: string; creationTimestamp: string}}>}>();
        const onLoad = jest.fn();
        const onChange = jest.fn();
        const watch = jest.fn(() => EMPTY);
        const listWatch = new ListWatch(
            () => request.promise,
            watch,
            onLoad,
            jest.fn(),
            onChange,
            jest.fn(),
            () => 0
        );

        listWatch.start();
        listWatch.stop();
        request.resolve({
            metadata: {resourceVersion: '1'},
            items: [{metadata: {name: 'old-workflow', namespace: 'namespace-a', creationTimestamp: '2026-01-01T00:00:00Z'}}]
        });
        await request.promise;

        expect(onLoad).not.toHaveBeenCalled();
        expect(onChange).not.toHaveBeenCalled();
        expect(watch).not.toHaveBeenCalled();
    });

    it('does not schedule a retry when a stopped list request fails', async () => {
        jest.useFakeTimers();
        const request = deferred<{metadata: {resourceVersion: string}; items: Array<{metadata: {name: string; namespace: string; creationTimestamp: string}}>}>();
        const onError = jest.fn();
        const listWatch = new ListWatch(
            () => request.promise,
            () => EMPTY,
            jest.fn(),
            jest.fn(),
            jest.fn(),
            onError,
            () => 0
        );

        listWatch.start();
        listWatch.stop();
        request.reject(new Error('Forbidden'));
        await request.promise.catch(() => undefined);

        expect(onError).not.toHaveBeenCalled();
        expect(jest.getTimerCount()).toBe(0);
    });
});
