import {act} from '@testing-library/react';

export interface MockXHR {
    open: jest.Mock;
    send: jest.Mock;
    abort: jest.Mock;
    upload: {addEventListener: jest.Mock; listeners: Record<string, (event: ProgressEvent) => void>};
    onload: (() => void) | null;
    onerror: (() => void) | null;
    status: number;
    statusText: string;
    responseText: string;
}

export function installMockXHR(): MockXHR {
    const listeners: Record<string, (event: ProgressEvent) => void> = {};
    const xhr: MockXHR = {
        open: jest.fn(),
        send: jest.fn(),
        abort: jest.fn(),
        upload: {
            addEventListener: jest.fn((event: string, cb: (event: ProgressEvent) => void) => {
                listeners[event] = cb;
            }),
            listeners
        },
        onload: null,
        onerror: null,
        status: 200,
        statusText: 'OK',
        responseText: ''
    };
    (window as unknown as {XMLHttpRequest: unknown}).XMLHttpRequest = jest.fn(() => xhr);
    return xhr;
}

export function makeFile(name = 'input.zip', contents = 'hello'): File {
    return new File([contents], name, {type: 'application/zip'});
}

export async function flush() {
    await act(async () => {
        await Promise.resolve();
    });
}
