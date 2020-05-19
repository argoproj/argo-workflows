import {Page} from 'argo-ui';
import * as React from 'react';
import {uiUrl} from '../../shared/base';

require('./user.scss');

export const Login = () => (
    <Page title='User' toolbar={{breadcrumbs: [{title: 'User'}]}}>
        <div className='argo-container'>
            <div className='white-box'>
                <h3>
                    <i className='fa fa-user-circle' /> User
                </h3>
                <p>
                    <button className='argo-button argo-button--base-o' onClick={() => (document.location.href = uiUrl('login'))}>
                        <i className='fa fa-sign-in-alt' /> Login
                    </button>
                </p>
            </div>
        </div>
    </Page>
);
