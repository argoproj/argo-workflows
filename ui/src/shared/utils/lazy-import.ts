import {ComponentType} from 'react';

/**
 * Lazy import wrapper that retries loading chunks on failure.
 * Fixes "Loading chunk X failed" errors by attempting to reload the chunk.
 * See: https://github.com/argoproj/argo-workflows/issues/15640
 */
export function lazyImport<T extends ComponentType<any>>(
    importFunc: () => Promise<{default: T}>,
    retries: number = 3,
    delayMs: number = 1000
): Promise<{default: T}> {
    return new Promise((resolve, reject) => {
        let attemptCount = 0;

        const attempt = () => {
            attemptCount++;
            importFunc()
                .then(resolve)
                .catch(error => {
                    if (attemptCount > retries) {
                        reject(error);
                        return;
                    }
                    const delay = delayMs * attemptCount;
                    setTimeout(attempt, delay);
                });
        };

        attempt();
    });
}
