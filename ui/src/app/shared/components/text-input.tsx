import * as React from 'react';

export const TextInput = ({placeholder, value, onChange, readOnly}: {value: string; onChange: (value: string) => void; readOnly?: boolean; placeholder?: string}) =>
    readOnly ? <>{value}</> : <input className='argo-field' value={value} onChange={e => onChange(e.target.value)} placeholder={placeholder} />;
