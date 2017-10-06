import { Observable, Observer } from 'rxjs';
import * as JSONStream from 'json-stream';
import * as shell from 'shelljs';
import * as shellEscape from 'shell-escape';

export function reactifyStream(stream, converter = item => item) {
    return new Observable((observer: Observer<any>) => {
        stream.on('data', (d) => observer.next(converter(d)));
        stream.on('end', () => observer.complete());
        stream.on('error', e => observer.error(e));
    });
}

export function reactifyStringStream(stream) {
    return reactifyStream(stream, item => item.toString());
}

export function reactifyJsonStream(stream) {
    return reactifyStream(stream.pipe(new JSONStream()), item => item);
}

export function exec(cmd: string[], rejectOnFail = true): Promise<{code, stdout, stderr}> {
    return new Promise((resolve, reject) => {
        shell.exec(shellEscape(cmd), { silent: true } , (code, stdout, stderr) => {
            let res = { code, stdout, stderr };
            if (code !== 0 && rejectOnFail) {
                reject(res);
            } else {
                resolve(res);
            }
        });
    });
}

export function timeout(milliseconds: number): Promise<any> {
    return new Promise(resolve => setTimeout(() => {
        resolve(true);
    }, milliseconds));
}

export async function execute(action: () => Promise<any>, retryCount: number, retryTimeoutMs: number, doNotFail = false) {
    let done = false;
    let error;
    while (!done && retryCount > 0) {
        try {
            error = null;
            await action();
            done = true;
        } catch (e) {
            error = e;
            retryCount--;
            await timeout(retryTimeoutMs);
        }
    }
    if (error && doNotFail === false) {
        throw error;
    }
}

export async function executeSafe(action: () => Promise<any>, retryCount: number, retryTimeoutMs: number, doNotFail = true) {
    return execute(action, retryCount, retryTimeoutMs, true);
}
