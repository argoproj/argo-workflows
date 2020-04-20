import {Page} from 'argo-ui';
import * as React from 'react';
import {uiUrl} from '../../shared/base';

require('./login.scss');

const getToken = () => {
    for (const cookie of document.cookie.split(';')) {
        if (cookie.startsWith('authorization=')) {
            return cookie.substring(14);
        }
    }
    return null;
};

const maybeLoggedIn = () => !!getToken();
const logout = () => {
    document.cookie = 'authorization=;Max-Age=0';
    document.location.reload(true);
};
const login = (token: string) => {
    document.cookie = 'authorization=' + token + ';SameSite=Strict';
    document.location.href = uiUrl('');
};
export const Login = () => (
    <Page title='Login' toolbar={{breadcrumbs: [{title: 'Login'}]}}>
        <div className='argo-container'>
            <p>
                <i className='fa fa-info-circle' /> You appear to be <b>logged {maybeLoggedIn() ? 'in' : 'out'}</b>. It may not be necessary to login to use Argo, it depends on how
                it is configured.
            </p>
            <p>
                Get your token using <code>argo auth token</code> and paste in this box.
            </p>
            <textarea id='token' cols={100} rows={20} />
            <div>
                {maybeLoggedIn() && (
                    <button className='argo-button argo-button--base-o' onClick={() => logout()}>
                        <i className='fa fa-lock' /> Logout
                    </button>
                )}
                <button className='argo-button argo-button--base-o' onClick={() => login((document.getElementById('token') as HTMLInputElement).value)}>
                    <i className='fa fa-lock-open' /> Login
                </button>
            </div>
        </div>
    </Page>
);
