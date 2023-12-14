import classNames from 'classnames';
import React, {ReactNode, useEffect, useRef, useState} from 'react';
import {createPortal} from 'react-dom';

import './dropdown.scss';

export interface DropDownProps {
    isMenu?: boolean;
    anchor: JSX.Element;
    children: ReactNode;
}

export function DropDown({isMenu, anchor, children}: DropDownProps) {
    const [opened, setOpened] = useState(false);
    const [left, setLeft] = useState(0);
    const [top, setTop] = useState(0);
    const anchorRef = useRef<HTMLDivElement>(null);
    const contentRef = useRef<HTMLDivElement>(null);
    const anchorEl = anchorRef.current;
    const contentEl = contentRef.current;

    function refreshState() {
        const anchorHeight = anchorEl.offsetHeight + 2;
        const {top: anchorTop, left: anchorLeft} = anchorEl.getBoundingClientRect();
        const newState = {top, left, opened};

        // Set top position
        if (contentEl.offsetHeight + anchorTop + anchorHeight > window.innerHeight) {
            newState.top = anchorTop - contentEl.offsetHeight - 2;
        } else {
            newState.top = anchorTop + anchorHeight;
        }

        // Set left position
        if (contentEl.offsetWidth + anchorLeft > window.innerWidth) {
            newState.left = anchorLeft - contentEl.offsetWidth + anchorEl.offsetWidth;
        } else {
            newState.left = anchorLeft;
        }

        return newState;
    }

    function open() {
        if (!contentEl || !anchorEl) {
            return;
        }

        const newState = refreshState();

        newState.opened = true;
        setOpened(newState.opened);
        setLeft(newState.left);
        setTop(newState.top);
    }

    function close(event: MouseEvent) {
        // Doesn't close when clicked inside the portal area
        if (contentEl.contains(event.target as Node) || anchorEl.contains(event.target as Node)) {
            return;
        }

        setOpened(false);
    }

    useEffect(() => {
        if (!opened) {
            return;
        }

        document.addEventListener('click', close);

        return () => {
            document.removeEventListener('click', close);
        };
    }, [opened]);

    return (
        <div className='argo-dropdown' ref={anchorRef}>
            <div
                className='argo-dropdown__anchor'
                onClick={event => {
                    open();
                    event.stopPropagation();
                }}>
                {anchor}
            </div>
            {createPortal(
                <div className={classNames('argo-dropdown__content', {opened, 'is-menu': isMenu})} style={{top, left}} ref={contentRef}>
                    {children}
                </div>,
                document.body
            )}
        </div>
    );
}
