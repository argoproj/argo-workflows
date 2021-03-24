import * as React from 'react';
import {ReactNode} from 'react';
import {Button} from './button';

export const LinkButton = (props: {to: string; children?: ReactNode}) => (
    <Button outline={true} onClick={() => (document.location.href = props.to)}>
        {props.children}
    </Button>
);
