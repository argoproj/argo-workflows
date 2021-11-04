import * as React from 'react';
import {Icon} from '../shared/components/icon';

export const AwesomeButton = ({icon, title}: {icon: Icon; title: string}) => {
    return (
        <a
            style={{
                position: 'relative',
                width: 150,
                height: 150,
                background: 'linear-gradient(#e66465, #9198e5)',
                borderRadius: 20,
                boxShadow: '1px 1px 1px gray',
                textAlign: 'center',
                verticalAlign: 'middle',
                margin: '10px',
                padding: '10px',
                display: 'inline-block',
                color: 'white'
            }}
            target='help'
            href='https://blog.argoproj.io/'>
            <div style={{fontSize: '28pt', lineHeight: '65px', verticalAlign: 'bottom'}}>
                <i className={'fa fa-' + icon} />
            </div>
            <div style={{fontSize: '14pt'}}>{title}</div>
        </a>
    );
};
