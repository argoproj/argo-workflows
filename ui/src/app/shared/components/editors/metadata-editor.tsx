import * as React from 'react';
import {kubernetes} from '../../../../models';
import {TextInput} from '../text-input';
import {KeyValueEditor} from './key-value-editor';

export const MetadataEditor = (props: {value: kubernetes.ObjectMeta; onChange: (value: kubernetes.ObjectMeta) => void}) => {
    return (
        <>
            <div className='white-box'>
                <div className='row white-box__details-row'>
                    <div className='columns small-4'>Name</div>
                    <div className='columns small-4'>
                        <TextInput onChange={name => props.onChange({...props.value, name})} value={props.value.name} readOnly={props.value.creationTimestamp !== null} />
                    </div>
                </div>
                <div className='row white-box__details-row'>
                    <div className='columns small-4'>Namespace</div>
                    <div className='columns small-4'>
                        <TextInput
                            onChange={namespace => props.onChange({...props.value, namespace})}
                            value={props.value.namespace}
                            readOnly={props.value.creationTimestamp !== null}
                        />
                    </div>
                </div>
            </div>
            <div className='white-box'>
                <h5>Labels</h5>
                <KeyValueEditor value={props.value.labels} onChange={labels => props.onChange({...props.value, labels})} />
            </div>
            <div className='white-box'>
                <h5>Annotations</h5>
                <KeyValueEditor
                    value={props.value.annotations}
                    onChange={annotations => props.onChange({...props.value, annotations})}
                    hide={key => key === 'kubectl.kubernetes.io/last-applied-configuration'}
                />
            </div>
        </>
    );
};
