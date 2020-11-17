import * as React from 'react';

export const TextInput = (props: {value: string; onChange: (value: string) => void; readOnly?: boolean; placeholder?: string}) =>
    props.readOnly ? <>{props.value}</> : <input className='argo-field' value={props.value} onChange={e => props.onChange(e.target.value)} placeholder={props.placeholder} />;
