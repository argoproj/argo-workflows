import * as React from 'react';
import {ReactNode} from 'react';
import {Button} from './button';

export const LinkButton = ({to, children}: {to: string; children?: ReactNode}) => (
    <Button outline={true} onClick={() => (document.location.href = to)}>
        {children}
    </Button>
);
