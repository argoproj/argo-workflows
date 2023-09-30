import classNames from 'classnames';
import React, {ReactNode, useEffect, useRef, useState} from 'react';
import {createPortal} from 'react-dom';

export interface DropDownProps {
    isMenu?: boolean;
    anchor: () => JSX.Element;
    children: ReactNode;
    qeId?: string;
}

export interface DropDownState {
    opened: boolean;
    left: number;
    top: number;
}

require('./dropdown.scss');

export function DropDown({isMenu, anchor, children, qeId}: DropDownProps) {
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
        if (!contentRef || !anchorRef) {
            return;
        }

        const newState = refreshState();

        newState.opened = true;
        setOpened(newState.opened);
        setLeft(newState.left);
        setTop(newState.top);
    }

    function close(event: MouseEvent) {
        if (opened) {
            // Doesn't close when clicked inside the portal area
            if (contentEl.contains(event.target as Node) || anchorEl.contains(event.target as Node)) {
                return;
            }

            setOpened(false);
        }
    }

    useEffect(() => {
        document.body.addEventListener('click', close);

        return () => document.body.removeEventListener('click', close);
    });

    return (
        <div className='argo-dropdown' ref={anchorRef}>
            <div
                qe-id={qeId}
                className='argo-dropdown__anchor'
                onClick={event => {
                    open();
                    event.stopPropagation();
                }}>
                {anchor()}
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
