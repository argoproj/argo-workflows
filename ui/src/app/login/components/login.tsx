import {Page} from 'argo-ui';
import * as React from 'react';

require('./login.scss');

const getToken = () => {
    const match = document.cookie.match(/Authorization=Bearer ([^ ]*)/);
    if (match && match.length >= 2) {
        return match[1];
    }
    return null;
};
const maybeLoggedIn = () => !!getToken();
const logout = () => {
    document.cookie = 'Authorization=; expires=Thu, 01 Jan 1970 00:00:00 UTC; path=/;';
    document.location.reload(true);
};
const login = (token: string) => {
    document.cookie = 'Authorization=Bearer ' + token + '; path=/;';
    document.location.href = '/workflows';
};
export const Login = () => (
    /* tslint:disable:max-line-length */
    <Page title='Login' toolbar={{breadcrumbs: [{title: 'Login'}]}}>
        <div className='row'>
            <div className='columns large-12 medium-12'>
                <p>Get your config using:</p>
                <div>
                    <code>kubectl config view --minify --raw -o json</code>
                </div>
                <p>Replace "localhost" or "12.0.0.1" with your hostname and paste below.</p>
                <p>
                    <label>Kubeconfig</label>
                </p>
                <div>
                    <textarea
                        id='kubeconfig'
                        cols={100}
                        rows={10}
                        onChange={event => {
                            const config = JSON.parse(event.target.value);
                            const restConfig = JSON.stringify({
                                host: config.clusters[0].cluster.server,
                                caData: config.clusters[0].cluster['certificate-authority-data'],
                                certData: config.users[0].user['client-certificate-data'],
                                keyData: config.users[0].user['client-key-data']
                            });
                            (document.getElementById('restConfig') as HTMLInputElement).value = restConfig;
                            (document.getElementById('token') as HTMLInputElement).value = btoa(restConfig);
                        }}
                    />
                </div>
                <p>
                    <label>REST Config</label>
                </p>
                <div>
                    <textarea id='restConfig' cols={100} rows={5} defaultValue='' />
                </div>
                <p>
                    <label>Bearer Token</label>
                </p>
                <div>
                    <textarea id='token' cols={100} rows={5} defaultValue={getToken()} />
                </div>
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
        </div>
    </Page>
);
