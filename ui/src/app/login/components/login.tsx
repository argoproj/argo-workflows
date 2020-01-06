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
    <Page title='Login' toolbar={{breadcrumbs: [{title: 'Login'}]}}>
        <div className='argo-container'>
            <p>
                <i className='fa fa-info-circle'/> You appear to be logged  {maybeLoggedIn() ? 'in' : 'out'}.
            </p>
            <p>
                Get your config using <code>argo token</code> and paste in this box.
            </p>
            <textarea id='token' cols={100} rows={20} defaultValue={getToken()} />
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
