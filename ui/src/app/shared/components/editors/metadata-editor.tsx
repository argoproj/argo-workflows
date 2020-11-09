import * as React from 'react';
import {kubernetes} from '../../../../models';
import {TextInput} from '../text-input';
import {KeyValueEditor} from './key-value-editor';

export const MetadataEditor = (props: {value: kubernetes.ObjectMeta; onChange: (value: kubernetes.ObjectMeta) => void}) => {
    const [metadata, setMetadata] = React.useState(props.value || {});
    React.useEffect(() => {
        props.onChange(metadata);
    }, [metadata]);
    return (
        <div key='metadata' className='white-box'>
            <h6>Metadata</h6>
            <label key='name'>
                Name
                <TextInput onChange={name => setMetadata({name})} value={metadata.name} readOnly={metadata.creationTimestamp !== null} />
            </label>
            <label key='namespace'>
                Namespace
                <TextInput onChange={namespace => setMetadata({namespace})} value={metadata.namespace} readOnly={metadata.creationTimestamp !== null} />
            </label>
            <label>
                Labels
                <KeyValueEditor value={props.value.labels} onChange={labels => setMetadata({labels})} />
            </label>
            <label>
                Annotations
                <KeyValueEditor
                    value={props.value.annotations}
                    onChange={annotations => setMetadata({annotations})}
                    hide={key => key === 'kubectl.kubernetes.io/last-applied-configuration'}
                />
            </label>
        </div>
    );
};
