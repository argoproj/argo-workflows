import * as React from 'react';
import {MouseEventHandler, ReactNode} from 'react';
import {Icon} from './icon';

type Type = 'Primary' | 'Secondary';

export const Button = ({
    onClick,
    children,
    title,
    type,
    icon,
    className
}: {
    onClick: MouseEventHandler;
    children?: ReactNode;
    title?: string;
    type?: Type;
    icon?: Icon;
    className?: string;
}) => (
    <button className={'argo-button ' + (type !== 'Secondary' ? 'argo-button--base' : 'argo-button--base-o') + ' ' + (className || '')} title={title} onClick={onClick}>
        {icon && <i className={'fa fa-' + icon} />} {children}
    </button>
);
