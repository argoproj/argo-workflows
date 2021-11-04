import * as React from 'react';

export const Modal = ({title, children}: {title: React.ReactNode; children: React.ReactNode}) => (
    <div style={{textAlign: 'center', verticalAlign: 'middle'}}>
        <h3>{title}</h3>
        {children}
    </div>
);
