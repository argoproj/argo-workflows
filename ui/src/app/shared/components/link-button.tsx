import * as React from 'react';
import {ReactNode} from 'react';

export const LinkButton = (props: {to: string; children?: ReactNode}) => (
    <button className='argo-button argo-button--base-o' onClick={() => (document.location.href = props.to)}>
        {props.children}
    </button>
);
