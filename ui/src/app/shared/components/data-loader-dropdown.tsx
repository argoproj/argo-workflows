import {DataLoader, Select, SelectOption} from 'argo-ui';
import * as React from 'react';
import {useState} from 'react';

export function DataLoaderDropdown(props: {load: () => Promise<(string | SelectOption)[]>; onChange: (value: string) => void; placeholder?: string}) {
    const [selected, setSelected] = useState('');

    return (
        <DataLoader load={props.load}>
            {list => (
                <Select
                    placeholder={props.placeholder}
                    options={list}
                    value={selected}
                    onChange={x => {
                        setSelected(x.value);
                        props.onChange(x.value);
                    }}
                />
            )}
        </DataLoader>
    );
}
