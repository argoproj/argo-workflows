import {Page} from 'argo-ui/src/components/page/page';
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
        <Page title='Login' toolbar={{breadcrumbs: [{title: 'Login'}]}}>
            <div className='argo-container'>
                <div className='white-box'>
                    <h3>
                        <i className='fa fa-shield-alt' /> Login
                    </h3>
                    <p>It may not be necessary to be logged in to use Argo Workflows, it depends on how it is configured.</p>
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
                            <a className='argo-button argo-button--base-o' href={uiUrlWithParams('oauth2/redirect', redirect)}>
                                <i className='fa fa-sign-in-alt' /> Login
                            </a>
                        </div>
                    </div>
                    <div className='columns small-4'>
                        <p>
                            If your organisation has configured <b>client authentication</b>, get your token following this instructions from{' '}
                            <a href='https://argo-workflows.readthedocs.io/en/latest/access-token/#token-creation'>here</a> and paste in this box:
                        </p>
                        <div>
                            <textarea id='token' cols={32} rows={8} value={token} onChange={e => setToken(e.target.value)} />
                        </div>
                        <div>
                            <a className='argo-button argo-button--base-o' href={uiUrl('')} onClick={() => setCookie('authorization', token)}>
                                <i className='fa fa-sign-in-alt' /> Login
                            </a>
                        </div>
                    </div>
                    <div className='columns small-4'>
                        <div>
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
                </div>
            </div>
        </Page>
    );
}
