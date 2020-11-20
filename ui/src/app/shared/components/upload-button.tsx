import * as React from 'react';
import {parse, stringify} from './object-parser';

export const UploadButton = <T extends any>(props: {lang?: string; onUpload: (value: T) => void; onError: (error: Error) => void}) => {
    const handleFiles = (files: FileList) => {
        files[0]
            .text()
            .then(value => stringify(parse(value), props.lang || 'yaml'))
            .then(value => props.onUpload(parse(value)))
            .then(() => props.onError(null))
            .catch(props.onError);
    };

    return (
        <label className='argo-button argo-button--base-o' key='upload-file'>
            <input type='file' onChange={e => handleFiles(e.target.files)} style={{display: 'none'}} />
            <i className='fa fa-upload' /> Upload file
        </label>
    );
};
