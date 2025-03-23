import {Page} from 'argo-ui/src/components/page/page';
import * as React from 'react';
import {useState} from 'react';

import {uiUrl, uiUrlWithParams} from '../shared/base';
import {services} from '../shared/services';
import {useCollectEvent} from '../shared/use-collect-event';

import './login.scss';

// Logout function
function logout() {
    document.cookie = 'authorization=;Max-Age=0';
    document.location.reload();
}

// Set auth token and redirect
function user(token: string) {
    if (!token || token.trim() === '') {
        alert('Please enter a valid token');
        return;
    }

    // Set the cookie
    const path = uiUrl('');
    document.cookie = 'authorization=' + token + ';SameSite=Strict;path=' + path;

    // Clear any cached API data to ensure fresh data after login
    services.info.clearCache();

    // Redirect to the main page
    document.location.href = path;
}

function getRedirect(): string {
    const urlParams = new URLSearchParams(new URL(document.location.href).search);
    if (urlParams.has('redirect')) {
        return 'redirect=' + urlParams.get('redirect');
    }
    return 'redirect=' + window.location.origin + '/workflows';
}

// Handle SSO login
function handleSsoLogin() {
    // Clear any cached API data to ensure fresh data after login
    services.info.clearCache();

    // Redirect to the SSO provider
    document.location.href = uiUrlWithParams('oauth2/redirect', [getRedirect()]);
}

export function Login() {
    useCollectEvent('openedLogin');
    const [token, setToken] = useState('');

    return (
        <Page title='Login' toolbar={{breadcrumbs: [{title: 'Login'}]}}>
            <div className='argo-container'>
                <div className='white-box'>
                    <h3>
                        <i className='fa fa-shield-alt' /> Login
                    </h3>
                    <p>It may not be necessary to be logged in to use Argo Workflows it depends on how it is configured.</p>
                    <p>
                        <a href='https://argo-workflows.readthedocs.io/en/latest/argo-server-auth-mode/'>Learn more</a>.
                    </p>
                </div>

                <div className='row'>
                    <div className='columns small-4'>
                        <p>
                            If your organisation has configured <b>single sign-on</b>:
                        </p>
                        <div>
                            <button className='argo-button argo-button--base-o' onClick={handleSsoLogin}>
                                <i className='fa fa-sign-in-alt' /> Login
                            </button>
                        </div>
                    </div>
                    <div className='columns small-4'>
                        <p>
                            If your organisation has configured <b>client authentication</b> get your token following this instructions from{' '}
                            <a href='https://argo-workflows.readthedocs.io/en/latest/access-token/#token-creation'>here</a> and paste in this box:
                        </p>
                        <div>
                            <textarea id='token' cols={32} rows={8} value={token} onChange={e => setToken(e.target.value)} />
                        </div>
                        <div>
                            <button className='argo-button argo-button--base-o' onClick={() => user(token)} disabled={!token || token.trim() === ''}>
                                <i className='fa fa-sign-in-alt' /> Login
                            </button>
                        </div>
                    </div>
                    <div className='columns small-4'>
                        <div>
                            <p>Something wrong? Try logging out and logging back in:</p>
                            <button className='argo-button argo-button--base-o' onClick={() => logout()}>
                                <i className='fa fa-sign-out-alt' /> Logout
                            </button>
                        </div>
                    </div>
                </div>
            </div>
        </Page>
    );
}
