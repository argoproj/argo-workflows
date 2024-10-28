import * as React from 'react';

import './modal.scss';

export const Modal = ({children, dismiss}: {children: React.ReactNode; dismiss: () => void}) => (
    <div className='modal'>
        <div className='modal-content'>
            <span className='modal-close' onClick={() => dismiss()}>
                &times;
            </span>
            {children}
        </div>
    </div>
);
