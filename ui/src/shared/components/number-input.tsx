import * as React from 'react';

import {TextInput} from './text-input';

export const NumberInput = ({onChange, value, placeholder, readOnly}: {value: number; onChange: (value: number | undefined) => void; readOnly?: boolean; placeholder?: string}) => {
    if (readOnly) {
        return <>{value != null && !isNaN(value) ? value : ''}</>;
    }
    return (
        <TextInput
            value={value != null && !isNaN(value) ? '' + value : ''}
            onChange={x => {
                const parsed = parseInt(x, 10);
                onChange(isNaN(parsed) ? undefined : parsed);
            }}
            placeholder={placeholder}
        />
    );
};
