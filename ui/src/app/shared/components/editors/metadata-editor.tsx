import * as React from 'react';
import {kubernetes} from '../../../../models';
import {TextInput} from '../text-input';
import {KeyValueEditor} from './key-value-editor';

export const MetadataEditor = (props: {value: kubernetes.ObjectMeta; onChange: (value: kubernetes.ObjectMeta) => void}) => {
    return (
        <div key='metadata' className='white-box'>
            <h5>Metadata</h5>
            <p key='name'>
                <label>Name</label>
                <TextInput onChange={name => props.onChange({...props.value, name})} value={props.value.name} readOnly={props.value.creationTimestamp !== null} />
            </p>
            <p key='namespace'>
                <label>Namespace</label>
                <TextInput onChange={namespace => props.onChange({...props.value, namespace})} value={props.value.namespace} readOnly={props.value.creationTimestamp !== null} />
            </p>
            <p key='labels'>
                <label>Labels</label>
                <KeyValueEditor value={props.value.labels} onChange={labels => props.onChange({...props.value, labels})} />
            </p>
            <p key='annotations'>
                <label>Annotations</label>
                <KeyValueEditor
                    value={props.value.annotations}
                    onChange={annotations => props.onChange({...props.value, annotations})}
                    hide={key => key === 'kubectl.kubernetes.io/last-applied-configuration'}
                />
            </p>
        </div>
    );
};
