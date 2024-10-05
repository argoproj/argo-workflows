import * as React from 'react';
import {useState} from 'react';
import {Notice} from './notice';

export function Nudge(props: React.PropsWithChildren<{key: string}>) {
    const [closed, setClosed] = useState(localStorage.getItem(props.key) !== null);
    function close() {
        setClosed(true);
        localStorage.setItem(props.key, '{}');
    }

    return (
        !closed && (
            <Notice style={{marginLeft: 0, marginRight: 0}}>
                {props.children}
                <span className='fa-pull-right'>
                    <a onClick={() => close()}>
                        <i className='fa fa-times' />
                    </a>{' '}
                </span>
            </Notice>
        )
    );
}
