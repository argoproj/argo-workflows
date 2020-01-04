import {Page} from 'argo-ui';
import * as React from 'react';

require('./login.scss');

const getToken = () => localStorage.getItem('token');
const maybeLoggedIn = () => !!getToken();
const logout = () => {
    localStorage.removeItem('token');
    document.location.reload(true);
};
const login = (token: string) => {
    localStorage.setItem('token', token);
    document.location.href = '/workflows';
};
export const Login = () => (
    /* tslint:disable:max-line-length */
    <Page title='Login' toolbar={{breadcrumbs: [{title: 'Login'}]}}>
        <div className='argo-container'>
            <p>
                Get your config using <code>argo token</code>.
            </p>
            <textarea id='token' cols={100} rows={10} defaultValue={getToken()} />
            <div>
                {maybeLoggedIn() && (
                    <button className='argo-button argo-button--base-o' onClick={() => logout()}>
                        Logout
                    </button>
                )}
                <button className='argo-button argo-button--base-o' onClick={() => login((document.getElementById('token') as HTMLInputElement).value)}>
                    Login
                </button>
            </div>
        </div>
    </Page>
);
