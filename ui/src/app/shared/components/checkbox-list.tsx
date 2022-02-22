import {Checkbox} from 'argo-ui/src/components/checkbox';
import * as React from 'react';

export const CheckboxList = ({onChange, values}: {values: {[label: string]: boolean}; onChange: (label: string, checked: boolean) => void}) => (
    <ul>
        {Object.entries(values)
            .sort()
            .map(([label, checked]) => (
                <li key={label}>
                    <label>
                        <Checkbox checked={checked} onChange={v => onChange(label, v)} /> {label || '-'}
                    </label>
                </li>
            ))}
    </ul>
);
