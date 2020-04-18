import {Page} from 'argo-ui';
import * as React from 'react';
import {uiUrl} from '../../shared/base';

require('./login.scss');

const getToken = () => {
    for (const cookie of document.cookie.split(';')) {
        if (cookie.startsWith('authorization=')) {
            return cookie.substring(14);
        }
    }
    return null;
};

const maybeLoggedIn = () => !!getToken();
const logout = () => {
    document.cookie = 'authorization=;';
    document.location.reload(true);
};
const login = (token: string) => {
    document.cookie = 'authorization=' + token + ';';
    document.location.href = uiUrl('');
};
export const Login = () => (
    <Page title='Login' toolbar={{breadcrumbs: [{title: 'Login'}]}}>
        <div className='argo-container'>
            <div className='white-box'>
                <h3>
                    <i className='fa fa-shield-alt' /> Login
                </h3>
                <p>
                    You appear to be <b>logged {maybeLoggedIn() ? 'in' : 'out'}</b>. It may not be necessary to login to use Argo, it depends on how it is configured.
                </p>
                <p>
                    <a href='https://github.com/argoproj/argo/blob/master/docs/auth.md'>Learn more</a>.
                </p>
            </div>

            <div className='row'>
                <div className='columns small-4'>
                    <p>If you're using single sign-on:</p>
                    <div>
                        <button className='argo-button argo-button--base-o' onClick={() => (document.location.href = uiUrl('oauth2/redirect'))}>
                            <i className='fa fa-sign-in-alt' /> Login
                        </button>
                    </div>
                </div>
                <div className='columns small-4'>
                    <p>
                        Otherwise, get your token using <code>argo auth token</code> and paste in this box:
                    </p>
                    <div>
                        <textarea id='token' cols={16} rows={8} defaultValue={getToken()} />
                    </div>
                    <div>
                        <button className='argo-button argo-button--base-o' onClick={() => login((document.getElementById('token') as HTMLInputElement).value)}>
                            <i className='fa fa-sign-in-alt' /> Login
                        </button>
                    </div>
                </div>
                <div className='columns small-4'>
                    <div>
                        <p>Something broken?</p>
                        <button className='argo-button argo-button--base-o' onClick={() => logout()}>
                            <i className='fa fa-sign-out-alt' /> Logout
                        </button>
                    </div>
                </div>
            </div>
        </div>
    </Page>
);
