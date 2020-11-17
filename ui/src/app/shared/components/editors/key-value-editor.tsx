import * as React from 'react';
import {TextInput} from '../text-input';

interface KeyValues {
    [key: string]: string;
}

export const KeyValueEditor = (props: {value: KeyValues; onChange: (value: KeyValues) => void; hide?: (key: string) => boolean}) => {
    const keyValues: KeyValues = props.value || {};
    const [name, setName] = React.useState('');
    const [value, setValue] = React.useState('');
    const deleteItem = (k: string) => {
        delete keyValues[k];
        props.onChange(keyValues);
    };
    const addItem = () => {
        keyValues[name] = value;
        props.onChange(keyValues);
    };
    return (
        <>
            {Object.entries(keyValues)
                .filter(([k]) => props.hide === undefined || !props.hide(k))
                .map(([k, v]) => (
                    <div className='row white-box__details-row' key={k}>
                        <div className='columns small-4'>{k}</div>
                        <div className='columns small-6'>{v}</div>
                        <div className='columns small-2'>
                            <button onClick={() => deleteItem(k)}>
                                <i className='fa fa-times-circle' />
                            </button>
                        </div>
                    </div>
                ))}
            <div className='row white-box__details-row'>
                <div className='columns small-4'>
                    <TextInput value={name} onChange={setName} placeholder='Name...' />
                </div>
                <div className='columns small-6'>
                    <TextInput value={value} onChange={setValue} placeholder='Value...' />
                </div>
                <div className='columns small-2'>
                    <button onClick={() => addItem()}>
                        <i className='fa fa-plus-circle' />
                    </button>
                </div>
            </div>
        </>
    );
};
