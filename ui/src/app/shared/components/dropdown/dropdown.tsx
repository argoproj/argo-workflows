import classNames from 'classnames';
import React, {ComponentType, ReactNode, useEffect, useRef} from 'react';
import {useState} from 'react';
import {createPortal} from 'react-dom';
import {BehaviorSubject, fromEvent, merge, Subscription} from 'rxjs';
import {filter} from 'rxjs/operators';

export interface DropDownProps {
    isMenu?: boolean;
    anchor: ComponentType;
    children: ReactNode;
    qeId?: string;
}

export interface DropDownState {
    opened: boolean;
    left: number;
    top: number;
}

require('./dropdown.scss');

const dropDownOpened = new BehaviorSubject<typeof DropDown>(null);

export function DropDown({isMenu, anchor: Anchnor, children, qeId}: DropDownProps) {
    const anchorRef = useRef<null | HTMLDivElement>(null);
    const contentRef = useRef<null | HTMLDivElement>(null);
    const subscriptionsRef = useRef<null | Subscription[]>(null);
    const [opened, setOpened] = useState(false);
    const [left, setLeft] = useState(0);
    const [top, setTop] = useState(0);

    useEffect(() => {
        const content = contentRef.current;
        const el = anchorRef.current;

        subscriptionsRef.current = [
            merge(
                dropDownOpened.pipe(filter(dropdown => dropdown !== DropDown)),
                fromEvent(document, 'click').pipe(
                    filter((event: Event) => {
                        return content && opened && !content.contains(event.target as Node) && !el.contains(event.target as Node);
                    })
                )
            ).subscribe(() => {
                close();
            }),
            fromEvent(document, 'scroll', {capture: true}).subscribe(() => {
                if (opened && content && el) {
                    const newState = refreshState();

                    setOpened(newState.opened);
                    setLeft(newState.left);
                    setTop(newState.top);
                }
            })
        ];

        return () => {
            (subscriptionsRef.current || []).forEach(s => s.unsubscribe());
            subscriptionsRef.current = [];
        };
    }, []);

    const refreshState = () => {
        const content = contentRef.current;
        const anchor = anchorRef.current;
        const anchorHeight = anchor.offsetHeight + 2;
        const {top: anchorTop, left: anchorLeft} = anchor.getBoundingClientRect();
        const newState = {top: anchorTop, left: anchorLeft, opened};

        // Set top position
        if (content.offsetHeight + anchorTop + anchorHeight > window.innerHeight) {
            newState.top = anchorTop - content.offsetHeight - 2;
        } else {
            newState.top = anchorTop + anchorHeight;
        }

        // Set left position
        if (content.offsetWidth + anchorLeft > window.innerWidth) {
            newState.left = anchorLeft - content.offsetWidth + anchor.offsetWidth;
        } else {
            newState.left = anchorLeft;
        }

        return newState;
    };

    const open = () => {
        if (!contentRef.current || !anchorRef.current) {
            return;
        }

        const newState = refreshState();

        newState.opened = true;
        setOpened(newState.opened);
        setLeft(newState.left);
        setTop(newState.top);
        dropDownOpened.next(DropDown);
    };

    const close = () => {
        setOpened(false);
    };

    return (
        <div className='argo-dropdown' ref={anchorRef}>
            <div
                qe-id={qeId}
                className='argo-dropdown__anchor'
                onClick={event => {
                    open();
                    event.stopPropagation();
                }}>
                <Anchnor />
            </div>
            {createPortal(
                <div className={classNames('argo-dropdown__content', {'opened': opened, 'is-menu': isMenu})} style={{top, left}} ref={contentRef}>
                    {children}
                </div>,
                document.body
            )}
        </div>
    );
}
