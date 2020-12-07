import * as React from 'react';
import {ReactNode, useState} from 'react';

require('./drop-down-button.scss');

export const DropDownButton = ({onClick, items, children}: {onClick: () => void; children: ReactNode; items: {value: string; onClick: () => void}[]}) => {
    const [dropped, setDropped] = useState(false);
    return (
        <div className='drop-down-button' onMouseEnter={() => setDropped(true)} onMouseLeave={() => setDropped(false)}>
            <button onClick={onClick} className='argo-button argo-button--base'>
                {children} <i className='fa fa-angle-down' />
            </button>
            <div className='items' style={{display: !dropped && 'none'}}>
                {items.map(option => (
                    <div key={option.value}>
                        <button className='argo-button argo-button--base item' onClick={option.onClick}>
                            {option.value}
                        </button>
                    </div>
                ))}
            </div>
        </div>
    );
};
