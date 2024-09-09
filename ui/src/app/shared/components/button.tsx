import * as React from 'react';
import {MouseEventHandler, ReactNode} from 'react';
import {Icon} from './icon';

export const Button = ({
    onClick,
    children,
    title,
    outline,
    icon,
    className,
    disabled
}: {
    onClick: MouseEventHandler;
    children?: ReactNode;
    title?: string;
    outline?: boolean;
    icon?: Icon;
    className?: string;
    disabled?: boolean;
}) => (
    <button
        style={{marginBottom: 2, marginRight: 2}}
        className={'argo-button ' + (!outline ? 'argo-button--base' : 'argo-button--base-o') + ' ' + (className || '')}
        title={title}
        onClick={onClick}
        disabled={disabled}>
        {icon && <i className={'fa fa-' + icon} />} {children}
    </button>
);
