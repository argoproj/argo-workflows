import * as React from 'react';
import {parse} from './object-parser';

export const UploadButton = <T extends any>(props: {onUpload: (value: T) => void; onError: (error: Error) => void}) => {
    const handleFiles = (files: FileList) => {
        files[0]
            .text()
            .then(value => props.onUpload(parse(value) as T))
            .catch(props.onError);
    };

    return (
        <label style={{marginBottom: 2, marginRight: 2}} className='argo-button argo-button--base-o' key='upload-file'>
            <input type='file' onChange={e => handleFiles(e.target.files)} style={{display: 'none'}} />
            <i className='fa fa-upload' /> Upload file
        </label>
    );
};
