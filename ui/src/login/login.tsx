import * as React from 'react';

import {uiUrl, uiUrlWithParams} from '../shared/base';
import {useCollectEvent} from '../shared/use-collect-event';

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
function getRedirect(): URLSearchParams {
    const urlParams = new URLSearchParams(document.location.search);
    return new URLSearchParams({redirect: urlParams.get('redirect') ?? '/workflows'});
}

export function Login() {
    useCollectEvent('openedLogin');
    return (
        <div className='login'>
            <div className='login__content show-for-large'>
                <div className='login__text'>Let&#39;s get workflows running!</div>
                <div className='argo__logo' />
            </div>
            <div className='login__box'>
                <div className='login__logo width-control'>
                    <img className='logo-image' src='assets/images/argo_o.svg' alt='argo' />
                </div>

                <div className='login__box-content'>
                    <div className='white-box login__info-section'>
                        <p>
                            It may not be necessary to be logged in to use Argo Workflows,
                            <br /> it depends on how it is configured.
                        </p>
                        <p>
                            <a href='https://argo-workflows.readthedocs.io/en/latest/argo-server-auth-mode/'>Learn more</a>.
                        </p>
                    </div>
                    <div className='white-box login__sso-section'>
                        <p>
                            If your organisation has configured <b>single sign-on</b>:
                        </p>
                        <div>
                            <button
                                className='argo-button argo-button--base-o'
                                onClick={() => {
                                    document.location.href = uiUrlWithParams('oauth2/redirect', getRedirect());
                                }}>
                                <i className='fa fa-sign-in-alt' /> Login
                            </button>
                        </div>
                    </div>
                    <div className='white-box login__token-section'>
                        <p>
                            If your organisation has configured <b>client authentication</b>,
                            <br />
                            get your token following this instructions from <a href='https://argo-workflows.readthedocs.io/en/latest/access-token/#token-creation'>here</a> and
                            <br />
                            paste in this box:
                        </p>
                        <div>
                            <textarea id='token' className='token-input' rows={4} />
                        </div>
                        <div>
                            <button className='argo-button argo-button--base-o' onClick={() => user((document.getElementById('token') as HTMLInputElement).value)}>
                                <i className='fa fa-sign-in-alt' /> Login
                            </button>
                        </div>
                    </div>
                    <div className='white-box login__logout-section'>
                        <p>Something wrong? Try logging out and logging back in:</p>
                        <button className='argo-button argo-button--base-o' onClick={logout}>
                            <i className='fa fa-sign-out-alt' /> Logout
                        </button>
                    </div>
                </div>
                <div className='login__footer'>
                    <a href='https://argoproj.io' target='_blank' rel='noreferrer'>
                        <img className='logo-image' src='assets/images/argologo.svg' alt='argo' />
                    </a>
                </div>
            </div>
        </div>
    );
}
