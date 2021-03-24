import * as React from 'react';
import {TextInput} from './text-input';

export const NumberInput = ({onChange, value, placeholder, readOnly}: {value: number; onChange: (value: number) => void; readOnly?: boolean; placeholder?: string}) =>
    readOnly ? <>{value}</> : <TextInput value={value != null ? '' + value : ''} onChange={x => onChange(parseInt(x, 10))} placeholder={placeholder} />;
