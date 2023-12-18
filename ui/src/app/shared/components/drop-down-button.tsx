import * as React from 'react';
import {ReactNode, useState} from 'react';

import {Button} from './button';

import './drop-down-button.scss';

export const DropDownButton = ({onClick, items, children}: {onClick: () => void; children: ReactNode; items: {value: string; onClick: () => void}[]}) => {
    const [dropped, setDropped] = useState(false);
    return (
        <div className='drop-down-button' onMouseEnter={() => setDropped(true)} onMouseLeave={() => setDropped(false)}>
            <Button onClick={onClick}>
                {children} <i className='fa fa-angle-down' />
            </Button>
            <div className='items' style={{display: !dropped && 'none'}}>
                {items.map(option => (
                    <div key={option.value}>
                        <Button className='item' onClick={option.onClick}>
                            {option.value}
                        </Button>
                    </div>
                ))}
            </div>
        </div>
    );
};
