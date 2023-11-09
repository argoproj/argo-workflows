import * as React from 'react';
import {CSSProperties, ReactNode} from 'react';

export const Notice = (props: {style?: CSSProperties; children: ReactNode}) => (
    <div className='white-box' style={{padding: 20, margin: 20, ...props.style}}>
        {props.children}
    </div>
);
