import {Page} from 'argo-ui/src/components/page/page';
import * as React from 'react';
import {useState} from 'react';

import {uiUrl, uiUrlWithParams} from '../shared/base';
import {services} from '../shared/services';
import {useCollectEvent} from '../shared/use-collect-event';

import './login.scss';

// Clear any cached data on logout
function logout() {
    document.cookie = 'authorization=;Max-Age=0';
    // Clear any cached API data
    services.info.clearCache();
    // Add a small delay before reload to ensure cookies are cleared
    setTimeout(() => {
        document.location.reload();
    }, 100);
}

// Set auth token and redirect with loading indicator
function user(token: string) {
    if (!token || token.trim() === '') {
        alert('Please enter a valid token');
        return;
    }

    // Show loading indicator
    const loadingElement = document.createElement('div');
    loadingElement.className = 'loading-container';
    loadingElement.innerHTML = `
        <div style="text-align: center">
            <div class="loading-spinner"></div>
            <p>Logging in...</p>
        </div>
    `;
    document.body.appendChild(loadingElement);

    // Set the cookie
    const path = uiUrl('');
    document.cookie = 'authorization=' + token + ';SameSite=Strict;path=' + path;

    // Clear any cached API data
    services.info.clearCache();

    // Add a small delay before redirect to ensure cookie is set
    setTimeout(() => {
        document.location.href = path;
    }, 300);
}

function getRedirect(): string {
    const urlParams = new URLSearchParams(new URL(document.location.href).search);
    if (urlParams.has('redirect')) {
        return 'redirect=' + urlParams.get('redirect');
    }
    return 'redirect=' + window.location.origin + '/workflows';
}

// Handle SSO login with loading indicator
function handleSsoLogin() {
    // Show loading indicator
    const loadingElement = document.createElement('div');
    loadingElement.className = 'loading-container';
    loadingElement.innerHTML = `
        <div style="text-align: center">
            <div class="loading-spinner"></div>
            <p>Redirecting to login provider...</p>
        </div>
    `;
    document.body.appendChild(loadingElement);

    // Clear any cached API data
    services.info.clearCache();

    // Add a small delay before redirect
    setTimeout(() => {
        document.location.href = uiUrlWithParams('oauth2/redirect', [getRedirect()]);
    }, 300);
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
