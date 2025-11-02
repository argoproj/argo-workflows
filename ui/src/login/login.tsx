import * as React from 'react';
import {useState} from 'react';
import {RouteComponentProps} from 'react-router';

import {uiUrl, uiUrlWithParams} from '../shared/base';
import {deleteCookie, setCookie} from '../shared/cookie';
import {useCollectEvent} from '../shared/use-collect-event';

import './login.scss';

export function Login({location, history}: RouteComponentProps<any>) {
    const urlParams = new URLSearchParams(location.search);
    const redirect = new URLSearchParams({redirect: urlParams.get('redirect') ?? uiUrl('workflows')});
    const [token, setToken] = useState('');
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
                            <a className='argo-button argo-button--base-o' href={uiUrlWithParams('oauth2/redirect', redirect)}>
                                <i className='fa fa-sign-in-alt' /> Login
                            </a>
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
                            <textarea id='token' className='token-input' rows={4} value={token} onChange={e => setToken(e.target.value)} />
                        </div>
                        <div>
                            <a className='argo-button argo-button--base-o' href={uiUrl('')} onClick={() => setCookie('authorization', token)}>
                                <i className='fa fa-sign-in-alt' /> Login
                            </a>
                        </div>
                    </div>
                    <div className='white-box login__logout-section'>
                        <p>Something wrong? Try logging out and logging back in:</p>
                        <a
                            className='argo-button argo-button--base-o'
                            onClick={() => {
                                deleteCookie('authorization');
                                history.go(0);
                            }}>
                            <i className='fa fa-sign-out-alt' /> Logout
                        </a>
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
