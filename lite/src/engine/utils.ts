import { Observable, Observer } from 'rxjs';
import * as shell from 'shelljs';
import * as shellEscape from 'shell-escape';
import { Docker } from 'node-docker-api';

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

export async function executeSafe(action: () => Promise<any>, retryCount: number = 1, retryTimeoutMs: number = 0, doNotFail = true) {
    return execute(action, retryCount, retryTimeoutMs, true);
}

export async function ensureImageExists(docker: Docker, imageUrl: string): Promise<any> {
    let res = await docker.image.list({filter: imageUrl});
    if (res.length === 0) {
        await exec(['docker', 'pull', imageUrl]);
    }
}
