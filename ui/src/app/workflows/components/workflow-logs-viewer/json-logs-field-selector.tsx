import * as React from 'react';
import {useState} from 'react';
import {TextInput} from '../../../shared/components/text-input';

export interface SelectedJsonFields {
    values: string[];
}

export function JsonLogsFieldSelector({fields, onChange}: {fields: SelectedJsonFields; onChange: (v: string[]) => void}) {
    const [inputFields, setInputFields] = useState(fields);
    const [key, setKey] = useState('');

    function deleteItem(k: string) {
        const index = inputFields.values.indexOf(k, 0);
        if (index === -1) {
            return;
        }
        const values = inputFields.values.filter(v => v !== k);
        setInputFields({values});
        onChange(values);
    }
    function addItem() {
        if (!key || key.trim().length === 0) {
            return;
        }
        const index = inputFields.values.indexOf(key, 0);
        if (index !== -1) {
            return;
        }
        const values = [...inputFields.values, key];
        setInputFields({values});
        setKey('');
        onChange(values);
    }

    return (
        <>
            {inputFields.values.map(k => (
                <div className='row white-box__details-row' key={k}>
                    <div className='columns small-10'>{k}</div>
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
                <div className='columns small-10'>
                    <TextInput value={key} onChange={setKey} placeholder='jsonPayload.message' />
                </div>
                <div className='columns small-2'>
                    <button onClick={addItem}>
                        <i className='fa fa-plus-circle' />
                    </button>
                </div>
            </div>
        </>
    );
}

export function extractJsonValue(obj: any, jsonpath: string): string | null {
    const fields = jsonpath.split('.');
    try {
        let target = obj;
        for (const field of fields) {
            target = target[field];
        }
        return typeof target === 'string' ? target : JSON.stringify(target);
    } catch (e) {
        return null;
    }
}
