import * as React from 'react';
import {ReactNode} from 'react';

import './toolbar-actions.scss';

export interface ToolbarAction {
    title: string;
    iconClassName?: string;
    disabled?: boolean;
    action?: () => any;
    // when set, the item is rendered as an anchor instead of a button, so that the browser's native link behaviour (e.g. "Open in new tab") works
    href?: string;
    target?: string;
}

// ToolbarActions is a replacement for the `actionMenu` of a `Page` toolbar, to be passed as the toolbar's `tools` instead.
// argo-ui's action menu can only render buttons, whereas this also renders links as anchors, with the same appearance.
// Any children are rendered on the right of the toolbar, where the tools normally are.
export function ToolbarActions({items, children}: {items: ToolbarAction[]; children?: ReactNode}) {
    return (
        <div className='top-bar__actions'>
            <div>
                {items.map((item, i) => {
                    const props = {className: 'argo-button argo-button--base', style: {marginRight: 2}};
                    const content = (
                        <>
                            {item.iconClassName && <i className={item.iconClassName} style={{marginLeft: '-5px', marginRight: '5px'}} />}
                            {item.title}
                        </>
                    );
                    return item.href !== undefined ? (
                        <a {...props} href={item.href} target={item.target} rel='noreferrer' key={i}>
                            {content}
                        </a>
                    ) : (
                        <button {...props} disabled={!!item.disabled} onClick={() => item.action()} key={i}>
                            {content}
                        </button>
                    );
                })}
            </div>
            <div>{children}</div>
        </div>
    );
}
