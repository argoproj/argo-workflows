import * as React from 'react';
import {Icon} from './icon';

export const BigButton = ({icon, title, onClick, href}: {icon: Icon; title: string; onClick?: () => void; href?: string}) => {
    return (
        <a
            style={{
                position: 'relative',
                width: 150,
                height: 150,
                background: 'linear-gradient(#e66465, #9198e5)',
                borderRadius: 20,
                textAlign: 'center',
                verticalAlign: 'middle',
                margin: '10px',
                padding: '10px',
                display: 'inline-block',
                color: 'white'
            }}
            target='_blank'
            onClick={onClick}
            href={href}
            rel='noreferrer'>
            <div style={{fontSize: '28pt', lineHeight: '65px', verticalAlign: 'bottom'}}>
                <i className={'fa fa-' + icon} />
            </div>
            <div style={{fontSize: '14pt'}}>{title}</div>
        </a>
    );
};
