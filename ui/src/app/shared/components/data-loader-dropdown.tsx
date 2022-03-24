import {DataLoader, Select, SelectOption} from 'argo-ui';
import * as React from 'react';

export const DataLoaderDropdown = ({
    load,
    placeholder,
    value,
    onChange
}: {
    load: () => Promise<(string | SelectOption)[]>;
    value: string;
    onChange: (value: string) => void;
    placeholder?: string;
}) => {
    return (
        <DataLoader noLoaderOnInputChange={true} load={load}>
            {list => <Select placeholder={placeholder} options={list} value={value} onChange={x => onChange(x.value)} />}
        </DataLoader>
    );
};
