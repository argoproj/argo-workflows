import {Checkbox} from 'argo-ui/src/components/checkbox';
import * as React from 'react';

export const CheckboxList = (props: {values: {[label: string]: boolean}; onChange: (label: string, checked: boolean) => void}) => (
    <ul>
        {Object.entries(props.values)
            .sort()
            .map(([label, checked]) => (
                <li key={label}>
                    <label>
                        <Checkbox checked={checked} onChange={v => props.onChange(label, v)} /> {label || '-'}
                    </label>
                </li>
            ))}
    </ul>
);
