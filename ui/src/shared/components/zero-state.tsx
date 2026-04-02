import * as React from 'react';
import {ReactNode} from 'react';

// https://designsystem.quickbooks.com/pattern/zero-states/
export const ZeroState = (props: {title?: string; children: ReactNode}) => (
    <div className='white-box' style={{margin: 20}}>
        <h4>{props.title || 'Nothing to show'}</h4>
        {props.children}
    </div>
);
