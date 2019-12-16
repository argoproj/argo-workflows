import * as express from 'express';
import {Observable, Observer} from 'rxjs';

export function reactifyStream(stream, converter = (item) => item) {
    return new Observable((observer: Observer < any >) => {
        stream.on('data', (d) => observer.next(converter(d)));
        stream.on('end', () => observer.complete());
        stream.on('error', (e) => observer.error(e));
    });
}

export function reactifyStringStream(stream) {
    return reactifyStream(stream, (item) => item.toString());
}

export function streamServerEvents <T>(req: express.Request, res: express.Response, source: Observable <T>, formatter: (input: T) => string) {
    res.setHeader('Content-Type', 'text/event-stream');
    res.setHeader('Transfer-Encoding', 'chunked');
    res.setHeader('X-Content-Type-Options', 'nosniff');

    const subscription = source.subscribe((info) => res.write(`data:${formatter(info)}\n\n`), (err) => {
        res.set(200);
        res.end();
    }, () => {
        res.set(200);
        res.end();
    });
    req.on('close', () => subscription.unsubscribe());
}

export function decodeBase64(input: string) {
    return new Buffer(input, 'base64').toString('ascii');
}

export function encodeBase64(input: string) {
    return new Buffer(input).toString('base64');
}
