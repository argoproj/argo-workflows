import {Page} from 'argo-ui';
import * as React from 'react';

require('./login.scss');

const getToken = () => localStorage.getItem('token');
const maybeLoggedIn = () => getToken() !== null;
const logout = () => localStorage.removeItem('token');
const login = (token: string) => localStorage.setItem('token', token);

export const Login = () => (
    <Page title='Login'>
        <div className='row'>
            <div className='columns large-12 medium-12'>
                <h1>Login</h1>
                <p>Copy and paste your kubeconfig base 64 encoded, e.g. </p>
                <div>
                    <code>kubectl config view | base64</code>
                </div>
                <div>
                    <textarea id='token' cols={200} rows={20} defaultValue={getToken()} />
                </div>
                <div>
                    {maybeLoggedIn && (
                        <button className='argo-button argo-button--base-o' onClick={() => logout()}>
                            Logout
                        </button>
                    )}
                    <button className='argo-button argo-button--base-o' onClick={() => login((document.getElementById('token') as HTMLInputElement).value)}>
                        Login
                    </button>
                </div>
            </div>
        </div>
    </Page>
);
