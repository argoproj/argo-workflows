import {DataLoader, Select, SelectOption} from 'argo-ui';
import * as React from 'react';

export const DataLoaderDropdown = (props: {load: () => Promise<(string | SelectOption)[]>; onChange: (value: string) => void}) => {
    const [selected, setSelected] = React.useState('');

    return (
        <DataLoader load={props.load}>
            {list => (
                <Select
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
};
