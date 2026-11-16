import {act, fireEvent, render, screen} from '@testing-library/react';
import React from 'react';

import {ArtifactsInput, ArtifactUploadResponse} from './artifacts-input';
import {flush, installMockXHR, makeFile} from './test-utils';

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

    it('POSTs to the upload endpoint prefixed with the configured base href', async () => {
        const base = document.createElement('base');
        base.setAttribute('href', '/argo/');
        document.head.appendChild(base);

        const xhr = installMockXHR();
        render(<ArtifactsInput {...defaultProps} />);

        const fileInput = document.querySelector('input[type="file"]') as HTMLInputElement;
        fireEvent.change(fileInput, {target: {files: [makeFile()]}});

        expect(xhr.open).toHaveBeenCalledWith('POST', '/argo/upload-artifacts/argo/my-template/input-artifact');

        document.head.removeChild(base);
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

    it('adds clusterScope=true to the upload URL for a ClusterWorkflowTemplate', () => {
        const xhr = installMockXHR();
        render(<ArtifactsInput {...defaultProps} clusterScope />);

        const fileInput = document.querySelector('input[type="file"]') as HTMLInputElement;
        fireEvent.change(fileInput, {target: {files: [makeFile()]}});

        expect(xhr.open).toHaveBeenCalledWith('POST', '/upload-artifacts/argo/my-template/input-artifact?clusterScope=true');
    });

    it('aborts an in-flight upload when unmounted without reporting a result', async () => {
        const xhr = installMockXHR();
        const onUploadComplete = jest.fn();
        const onError = jest.fn();
        const {unmount} = render(<ArtifactsInput {...defaultProps} onUploadComplete={onUploadComplete} onError={onError} />);

        const fileInput = document.querySelector('input[type="file"]') as HTMLInputElement;
        fireEvent.change(fileInput, {target: {files: [makeFile()]}});

        unmount();
        await flush();

        expect(xhr.abort).toHaveBeenCalledTimes(1);
        expect(onUploadComplete).not.toHaveBeenCalled();
        expect(onError).not.toHaveBeenCalled();
    });

    it('does not abort when the upload has already completed', async () => {
        const xhr = installMockXHR();
        const {unmount} = render(<ArtifactsInput {...defaultProps} />);

        const fileInput = document.querySelector('input[type="file"]') as HTMLInputElement;
        fireEvent.change(fileInput, {target: {files: [makeFile()]}});

        xhr.status = 201;
        xhr.responseText = JSON.stringify({name: 'input-artifact', key: 'uploads/argo/abc/input.zip'});
        act(() => {
            xhr.onload?.();
        });
        await flush();
        unmount();

        expect(xhr.abort).not.toHaveBeenCalled();
    });

    it('does not abort when no upload is in flight', () => {
        const xhr = installMockXHR();
        const {unmount} = render(<ArtifactsInput {...defaultProps} />);

        unmount();

        expect(xhr.abort).not.toHaveBeenCalled();
    });
});
