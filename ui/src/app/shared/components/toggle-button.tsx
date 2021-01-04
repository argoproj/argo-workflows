import * as React from 'react';
import {ReactNode} from 'react';
import {Button} from './button';

export const ToggleButton = ({title, children, onToggle, toggled}: {toggled: boolean; onToggle: () => void; children: ReactNode; title?: string}) => (
    <Button onClick={() => onToggle()} type='Secondary' title={title} icon={toggled ? 'toggle-on' : 'toggle-off'}>
        {children}
    </Button>
);
