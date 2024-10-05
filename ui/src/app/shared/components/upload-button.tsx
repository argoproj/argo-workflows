import * as React from 'react';
import {parse} from './object-parser';

export function UploadButton<T>(props: {onUpload: (value: T) => void; onError: (error: Error) => void}) {
    async function handleFiles(files: FileList) {
        try {
            const value = await files[0].text();
            props.onUpload(parse(value) as T);
        } catch (err) {
            props.onError(err);
        }
    }

    return (
        <label style={{marginBottom: 2, marginRight: 2}} className='argo-button argo-button--base-o' key='upload-file'>
            <input type='file' onChange={e => handleFiles(e.target.files)} style={{display: 'none'}} />
            <i className='fa fa-upload' /> Upload file
        </label>
    );
}
