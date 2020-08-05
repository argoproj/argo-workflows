import * as React from 'react';
import {ReactNode} from 'react';
import {Link} from 'react-router-dom';

export const LinkButton = (props: {to: string; children?: ReactNode}) => (
    <Link className='argo-button argo-button--base' to={props.to}>
        {props.children}
    </Link>
);
