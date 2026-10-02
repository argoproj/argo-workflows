import * as React from 'react';
import {MouseEventHandler, ReactNode} from 'react';

import {Icon} from './icon';

export const Button = ({
    onClick,
    href,
    target,
    children,
    title,
    outline,
    icon,
    className
}: {
    onClick?: MouseEventHandler;
    // when set, renders an anchor styled as a button, so that the browser's native link behaviour (e.g. "Open in new tab") works
    href?: string;
    target?: string;
    children?: ReactNode;
    title?: string;
    outline?: boolean;
    icon?: Icon;
    className?: string;
}) => {
    const props = {
        style: {marginBottom: 2, marginRight: 2},
        className: 'argo-button ' + (!outline ? 'argo-button--base' : 'argo-button--base-o') + ' ' + (className || ''),
        title,
        onClick
    };
    const content = (
        <>
            {icon && <i className={'fa fa-' + icon} />} {children}
        </>
    );
    if (href !== undefined) {
        return (
            <a {...props} href={href} target={target} rel='noreferrer'>
                {content}
            </a>
        );
    }
    return <button {...props}>{content}</button>;
};
