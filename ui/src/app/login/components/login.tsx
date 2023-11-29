import {Page} from 'argo-ui';
import * as React from 'react';

import {uiUrl, uiUrlWithParams} from '../../shared/base';
import {useCollectEvent} from '../../shared/components/use-collect-event';

import './login.scss';

function logout() {
    document.cookie = 'authorization=;Max-Age=0';
    document.location.reload();
}
function user(token: string) {
    const path = uiUrl('');
    document.cookie = 'authorization=' + token + ';SameSite=Strict;path=' + path;
    document.location.href = path;
}
function getRedirect(): string {
    const urlParams = new URLSearchParams(new URL(document.location.href).search);
    if (urlParams.has('redirect')) {
        return 'redirect=' + urlParams.get('redirect');
    }
    return 'redirect=' + window.location.origin + '/workflows';
}

export function Login() {
    useCollectEvent('openedLogin');
    return (
        <Page title='Login' toolbar={{breadcrumbs: [{title: 'Login'}]}}>
            <div className='argo-container'>
                <div className='white-box'>
                    <h3>
                        <i className='fa fa-shield-alt' /> Login
                    </h3>
                    <p>It may not be necessary to be logged in to use Argo Workflows, it depends on how it is configured.</p>
                    <p>
                        <a href='https://argoproj.github.io/argo-workflows/argo-server-auth-mode/'>Learn more</a>.
                    </p>
                </div>

                <div className='row'>
                    <div className='columns small-4'>
                        <p>
                            If your organisation has configured <b>single sign-on</b>:
                        </p>
                        <div>
                            <button
                                className='argo-button argo-button--base-o'
                                onClick={() => {
                                    document.location.href = uiUrlWithParams('oauth2/redirect', [getRedirect()]);
                                }}>
                                <i className='fa fa-sign-in-alt' /> Login
                            </button>
                        </div>
                    </div>
                    <div className='columns small-4'>
                        <p>
                            If your organisation has configured <b>client authentication</b>, get your token following this instructions from{' '}
                            <a href='https://argoproj.github.io/argo-workflows/access-token/#token-creation'>here</a> and paste in this box:
                        </p>
                        <div>
                            <textarea id='token' cols={32} rows={8} />
                        </div>
                        <div>
                            <button className='argo-button argo-button--base-o' onClick={() => user((document.getElementById('token') as HTMLInputElement).value)}>
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
