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
        <div>
            {Object.entries(keyValues)
                .filter(([k]) => props.hide === undefined || !props.hide(k))
                .map(([k, v]) => (
                    <div key={k}>
                        {k}={v}{' '}
                        <button onClick={() => deleteItem(k)}>
                            <i className='fa fa-times-circle' />
                        </button>
                    </div>
                ))}
            <div key='new'>
                <TextInput value={name} onChange={setName} />
                <TextInput value={value} onChange={setValue} />
                <button onClick={() => addItem()}>
                    <i className='fa fa-plus-circle' />
                </button>
            </div>
        </div>
    );
};
