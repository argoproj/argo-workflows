import * as React from 'react';

export const TextInput = (props: {value: string; onChange: (value: string) => void; readOnly?: boolean}) => (
    <input value={props.value} className='argo-field' onChange={e => props.onChange(e.target.value)} readOnly={props.readOnly} />
);
