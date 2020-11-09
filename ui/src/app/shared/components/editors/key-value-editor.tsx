import * as React from 'react';
import {useState} from 'react';

interface KeyValues {
    [key: string]: string;
}

export const KeyValueEditor = (props: {value: KeyValues; onChange: (value: KeyValues) => void; hide?: (key: string) => boolean}) => {
    const [keyValues, setKeyValues] = useState(props.value || {});
    const deleteItem = (k: string) => {
        setKeyValues(s => {
            delete s[k];
            return s;
        });
    };
    const [name, setName] = React.useState('');
    const [value, setValue] = React.useState('');
    const addItem = () => {
        setKeyValues(s => {
            s[name] = value;
            return s;
        });
    };
    React.useEffect(() => {
        props.onChange(keyValues);
    }, [keyValues]);
    return (
        <div className='wf-row-labels ' style={{cursor: 'default'}}>
            {Object.entries(keyValues)
                .filter(([k]) => props.hide === undefined || !props.hide(k))
                .map(([k, v]) => (
                    <div className='tag' key={k}>
                        <div className='key'>{k}</div>
                        <div className='value'>
                            {v}{' '}
                            <button onClick={() => deleteItem(k)}>
                                <i className='fa fa-times-circle' />
                            </button>
                        </div>
                    </div>
                ))}
            <div className='tag' key='new'>
                <div className='key'>
                    <input value={name} onChange={e => setName(e.target.value)} />
                </div>
                <div className='value'>
                    <input value={value} onChange={e => setValue(e.target.value)} />{' '}
                    <button onClick={() => addItem()}>
                        <i className='fa fa-plus-circle' />
                    </button>
                </div>
            </div>
        </div>
    );
};
