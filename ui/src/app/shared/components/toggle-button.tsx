import * as React from 'react';
import {ReactNode} from 'react';
import {Button} from './button';

export const ToggleButton = (props: {toggled: boolean; onToggle: () => void; children: ReactNode; title?: string}) => (
    <Button onClick={() => props.onToggle} type='Secondary' title={props.title} icon={props.toggled ? 'toggle-on' : 'toggle-off'}>
        {props.children}
    </Button>
);
