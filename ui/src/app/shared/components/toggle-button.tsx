import * as React from 'react';
import {ReactNode} from 'react';

export const ToggleButton = (props: {toggled: boolean; onToggle: () => void; children: ReactNode; title?: string}) => (
    <button className='argo-button argo-button--base-o' title={props.title} onClick={() => props.onToggle()}>
        {props.toggled ? <i className='fa fa-toggle-on' /> : <i className='fa fa-toggle-off' />} {props.children}
    </button>
);
