import * as React from 'react';
import {useState, useRef} from 'react';

import './artifacts-input.scss';

interface ArtifactsInputProps {
    namespace: string;
    artifactName: string;
    onUploadComplete?: (response: ArtifactUploadResponse) => void;
    onError?: (error: Error) => void;
}

export interface ArtifactUploadResponse {
    name: string;
    key: string;
    location: any;
}

type UploadStatus = 'idle' | 'uploading' | 'success' | 'error';

export function ArtifactsInput({namespace, artifactName, onUploadComplete, onError}: ArtifactsInputProps) {
    const [status, setStatus] = useState<UploadStatus>('idle');
    const [progress, setProgress] = useState(0);
    const [fileName, setFileName] = useState<string | null>(null);
    const [errorMessage, setErrorMessage] = useState<string | null>(null);
    const fileInputRef = useRef<HTMLInputElement>(null);

    const handleFileSelect = () => {
        fileInputRef.current?.click();
    };

    const handleFileChange = async (event: React.ChangeEvent<HTMLInputElement>) => {
        const file = event.target.files?.[0];
        if (!file) {
            return;
        }

        setFileName(file.name);
        setStatus('uploading');
        setProgress(0);
        setErrorMessage(null);

        try {
            const formData = new FormData();
            formData.append('file', file);

            const xhr = new XMLHttpRequest();

            xhr.upload.addEventListener('progress', event => {
                if (event.lengthComputable) {
                    const percentComplete = Math.round((event.loaded / event.total) * 100);
                    setProgress(percentComplete);
                }
            });

            await new Promise<ArtifactUploadResponse>((resolve, reject) => {
                xhr.onload = () => {
                    if (xhr.status >= 200 && xhr.status < 300) {
                        try {
                            const response = JSON.parse(xhr.responseText) as ArtifactUploadResponse;
                            setStatus('success');
                            if (onUploadComplete) {
                                onUploadComplete(response);
                            }
                            resolve(response);
                        } catch (e) {
                            reject(new Error('Failed to parse response'));
                        }
                    } else {
                        reject(new Error(`Upload failed: ${xhr.statusText}`));
                    }
                };

                xhr.onerror = () => {
                    reject(new Error('Network error'));
                };

                xhr.open('POST', `/upload-artifacts/${namespace}/${artifactName}`);
                xhr.send(formData);
            });
        } catch (error) {
            setStatus('error');
            const errorMsg = error instanceof Error ? error.message : 'Unknown error';
            setErrorMessage(errorMsg);
            if (onError) {
                onError(error instanceof Error ? error : new Error(errorMsg));
            }
        }
    };

    const handleReset = () => {
        setStatus('idle');
        setProgress(0);
        setFileName(null);
        setErrorMessage(null);
        if (fileInputRef.current) {
            fileInputRef.current.value = '';
        }
    };

    return (
        <div className='artifacts-input'>
            <input ref={fileInputRef} type='file' onChange={handleFileChange} style={{display: 'none'}} />

            {status === 'idle' && (
                <div className='artifacts-input__dropzone' onClick={handleFileSelect}>
                    <i className='fa fa-upload' />
                    <span>Click to select a file or drag and drop</span>
                </div>
            )}

            {status === 'uploading' && (
                <div className='artifacts-input__progress'>
                    <div className='artifacts-input__file-name'>
                        <i className='fa fa-file' /> {fileName}
                    </div>
                    <div className='artifacts-input__progress-bar'>
                        <div className='artifacts-input__progress-fill' style={{width: `${progress}%`}} />
                    </div>
                    <div className='artifacts-input__progress-text'>{progress}%</div>
                </div>
            )}

            {status === 'success' && (
                <div className='artifacts-input__success'>
                    <i className='fa fa-check-circle' />
                    <span>{fileName} uploaded successfully</span>
                    <button className='argo-button argo-button--base-o' onClick={handleReset}>
                        Upload Another
                    </button>
                </div>
            )}

            {status === 'error' && (
                <div className='artifacts-input__error'>
                    <i className='fa fa-exclamation-circle' />
                    <span>{errorMessage}</span>
                    <button className='argo-button argo-button--base-o' onClick={handleReset}>
                        Try Again
                    </button>
                </div>
            )}
        </div>
    );
}
