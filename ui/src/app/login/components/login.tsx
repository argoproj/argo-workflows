import {Page} from 'argo-ui';
import * as React from 'react';

require('./login.scss');

const getToken = () => localStorage.getItem('token');
const maybeLoggedIn = () => getToken() !== null;
const logout = () => localStorage.removeItem('token');
const login = (token: string) => localStorage.setItem('token', token);
export const Login = () => (
    /* tslint:disable:max-line-length */
    <Page title='Login'>
        <div className='row'>
            <div className='columns large-12 medium-12'>
                <h1>Login</h1>
                <p>Get your config using:</p>
                <div>
                    <code>kubectl config view --minify --raw</code>
                </div>
                <p>
                    Extract the fields you need by following <a href='https://github.com/kubernetes/client-go/blob/master/tools/clientcmd/client_config.go#L127'>this code</a>,
                    e.g.:
                </p>
                <div>
                    <code>
                        {JSON.stringify(
                            {
                                Host: 'https://10.96.0.1:443',
                                BearerTokenFile: '/var/run/secrets/kubernetes.io/serviceaccount/token',
                                CAFile: '/var/run/secrets/kubernetes.io/serviceaccount/ca.crt'
                            },
                            null,
                            '  '
                        )}
                    </code>
                </div>
                <p>
                    Pipe via <code>base64</code>, e.g.:
                </p>
                <div>
                    <code>
                        eyJIb3N0IjoiaHR0cHM6Ly8xMC45Ni4wLjE6NDQzIiwiQVBJUGF0aCI6IiIsIkFjY2VwdENvbnRlbnRUeXBlcyI6IiIsIkNvbnRlbnRUeXBlIjoiIiwiR3JvdXBWZXJzaW9uIjpudWxsLCJOZWdvdGlhdGVkU2VyaWFsaXplciI6bnVsbCwiVXNlcm5hbWUiOiIiLCJQYXNzd29yZCI6IiIsIkJlYXJlclRva2VuIjoiIiwiQmVhcmVyVG9rZW5GaWxlIjoiL3Zhci9ydW4vc2VjcmV0cy9rdWJlcm5ldGVzLmlvL3NlcnZpY2VhY2NvdW50L3Rva2VuIiwiSW1wZXJzb25hdGUiOnsiVXNlck5hbWUiOiIiLCJHcm91cHMiOm51bGwsIkV4dHJhIjpudWxsfSwiQXV0aFByb3ZpZGVyIjpudWxsLCJJbnNlY3VyZSI6ZmFsc2UsIlNlcnZlck5hbWUiOiIiLCJDZXJ0RmlsZSI6IiIsIktleUZpbGUiOiIiLCJDQUZpbGUiOiIvdmFyL3J1bi9zZWNyZXRzL2t1YmVybmV0ZXMuaW8vc2VydmljZWFjY291bnQvY2EuY3J0IiwiQ2VydERhdGEiOm51bGwsIktleURhdGEiOm51bGwsIkNBRGF0YSI6bnVsbCwiVXNlckFnZW50IjoiIiwiUVBTIjowLCJCdXJzdCI6MCwiUmF0ZUxpbWl0ZXIiOm51bGwsIlRpbWVvdXQiOjB9
                    </code>
                </div>
                <p>Then paste below:</p>
                <div>
                    <textarea id='token' cols={100} rows={10} defaultValue={getToken()} />
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
