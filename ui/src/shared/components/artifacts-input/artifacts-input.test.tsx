import {act, fireEvent, render, screen} from '@testing-library/react';
import React from 'react';

import {ArtifactsInput, ArtifactUploadResponse} from './artifacts-input';

// Minimal XMLHttpRequest stub — captures open/send/onload/onerror so tests can
// simulate a completion (or a network error) at the moment they want to.
interface MockXHR {
    open: jest.Mock;
    send: jest.Mock;
    upload: {addEventListener: jest.Mock; listeners: Record<string, (event: ProgressEvent) => void>};
    onload: (() => void) | null;
    onerror: (() => void) | null;
    status: number;
    statusText: string;
    responseText: string;
}

function installMockXHR(): MockXHR {
    const listeners: Record<string, (event: ProgressEvent) => void> = {};
    const xhr: MockXHR = {
        open: jest.fn(),
        send: jest.fn(),
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

function makeFile(name = 'input.zip', contents = 'hello'): File {
    return new File([contents], name, {type: 'application/zip'});
}

async function flush() {
    // Let the queued promise inside uploadFile settle before assertions.
    await act(async () => {
        await Promise.resolve();
    });
}

describe('ArtifactsInput', () => {
    const defaultProps = {
        namespace: 'argo',
        workflowTemplateName: 'my-template',
        artifactName: 'input-artifact'
    };

    afterEach(() => {
        jest.restoreAllMocks();
    });

    it('POSTs to the upload endpoint with URL-encoded path parts and reports success', async () => {
        const xhr = installMockXHR();
        const onUploadComplete = jest.fn();
        const onUploadStart = jest.fn();

        render(<ArtifactsInput {...defaultProps} namespace='team/argo' onUploadStart={onUploadStart} onUploadComplete={onUploadComplete} />);

        const fileInput = document.querySelector('input[type="file"]') as HTMLInputElement;
        fireEvent.change(fileInput, {target: {files: [makeFile()]}});

        expect(onUploadStart).toHaveBeenCalledTimes(1);
        expect(xhr.open).toHaveBeenCalledWith(
            'POST',
            // 'team/argo' contains a '/' so the namespace path segment must be encoded.
            '/upload-artifacts/team%2Fargo/my-template/input-artifact'
        );
        expect(xhr.send).toHaveBeenCalledTimes(1);

        // Simulate a successful upload response.
        xhr.status = 201;
        xhr.responseText = JSON.stringify({name: 'input-artifact', key: 'uploads/argo/abc/input.zip'});
        act(() => {
            xhr.onload?.();
        });
        await flush();

        expect(onUploadComplete).toHaveBeenCalledTimes(1);
        const response: ArtifactUploadResponse = onUploadComplete.mock.calls[0][0];
        expect(response.key).toBe('uploads/argo/abc/input.zip');
        expect(screen.getByText(/uploaded successfully/i)).toBeTruthy();
    });

    it('updates progress from upload progress events', async () => {
        const xhr = installMockXHR();
        render(<ArtifactsInput {...defaultProps} />);

        const fileInput = document.querySelector('input[type="file"]') as HTMLInputElement;
        fireEvent.change(fileInput, {target: {files: [makeFile()]}});

        act(() => {
            xhr.upload.listeners.progress?.({lengthComputable: true, loaded: 250, total: 1000} as ProgressEvent);
        });

        expect(screen.getByText('25%')).toBeTruthy();
    });

    it('surfaces a non-2xx response as an error', async () => {
        const xhr = installMockXHR();
        const onError = jest.fn();
        render(<ArtifactsInput {...defaultProps} onError={onError} />);

        const fileInput = document.querySelector('input[type="file"]') as HTMLInputElement;
        fireEvent.change(fileInput, {target: {files: [makeFile()]}});

        xhr.status = 413;
        xhr.statusText = 'Request Entity Too Large';
        act(() => {
            xhr.onload?.();
        });
        await flush();

        expect(onError).toHaveBeenCalledTimes(1);
        expect(onError.mock.calls[0][0].message).toMatch(/Request Entity Too Large/);
        expect(screen.getByText(/Request Entity Too Large/)).toBeTruthy();
    });

    it('surfaces JSON parse failure on a 2xx response as an error', async () => {
        const xhr = installMockXHR();
        const onError = jest.fn();
        const onUploadComplete = jest.fn();
        render(<ArtifactsInput {...defaultProps} onError={onError} onUploadComplete={onUploadComplete} />);

        const fileInput = document.querySelector('input[type="file"]') as HTMLInputElement;
        fireEvent.change(fileInput, {target: {files: [makeFile()]}});

        xhr.status = 200;
        xhr.responseText = 'not-json';
        act(() => {
            xhr.onload?.();
        });
        await flush();

        expect(onUploadComplete).not.toHaveBeenCalled();
        expect(onError).toHaveBeenCalledTimes(1);
        expect(onError.mock.calls[0][0].message).toBe('Failed to parse response');
    });

    it('surfaces a network error as an error', async () => {
        const xhr = installMockXHR();
        const onError = jest.fn();
        render(<ArtifactsInput {...defaultProps} onError={onError} />);

        const fileInput = document.querySelector('input[type="file"]') as HTMLInputElement;
        fireEvent.change(fileInput, {target: {files: [makeFile()]}});

        act(() => {
            xhr.onerror?.();
        });
        await flush();

        expect(onError).toHaveBeenCalledTimes(1);
        expect(onError.mock.calls[0][0].message).toBe('Network error');
    });

    it('uploads a file dropped onto the dropzone', () => {
        const xhr = installMockXHR();
        render(<ArtifactsInput {...defaultProps} />);

        const dropzone = document.querySelector('.artifacts-input__dropzone') as HTMLElement;
        expect(dropzone).toBeTruthy();

        fireEvent.drop(dropzone, {dataTransfer: {files: [makeFile('dropped.zip')]}});

        expect(xhr.open).toHaveBeenCalledTimes(1);
        expect(xhr.send).toHaveBeenCalledTimes(1);
    });
});
