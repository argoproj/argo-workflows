import * as React from 'react';
import {useState} from 'react';

import {TextInput} from '../text-input';

interface KeyValues {
    [key: string]: string;
}

export function KeyValueEditor({onChange, keyValues, hide}: {keyValues: KeyValues; onChange: (value: KeyValues) => void; hide?: (key: string) => boolean}) {
    const [name, setName] = useState('');
    const [value, setValue] = useState('');

    function deleteItem(k: string) {
        delete keyValues[k];
        onChange(keyValues);
    }
    function addItem() {
        if (!name || !value) {
            return;
        }
        keyValues[name] = value;
        onChange(keyValues);
        setName('');
        setValue('');
    }

    return (
        <>
            {Object.entries(keyValues)
                .filter(([k]) => hide === undefined || !hide(k))
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
            <div
                className='row white-box__details-row'
                onKeyPress={e => {
                    if (e.key === 'Enter') {
                        addItem();
                    }
                }}>
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
}

KeyValueEditor.defaultProps = {
    keyValues: {}
};
