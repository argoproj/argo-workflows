import {Page} from 'argo-ui';
import * as React from 'react';
import {RouteComponentProps} from 'react-router';
import {User} from '../../../models';
import {uiUrl} from '../../shared/base';
import {services} from '../../shared/services';

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

interface State {
    error?: Error;
    user?: User;
}

export class Login extends React.Component<RouteComponentProps<any>, State> {
    constructor(props: RouteComponentProps<any>) {
        super(props);
        this.state = {};
    }
    public componentDidMount() {
        if (maybeLoggedIn()) {
            services.info
                .get()
                .then(info => this.setState({user: info.user}))
                .catch(error => this.setState({error}));
        }
    }

    private get username() {
        return this.state.user ? this.state.user.name : 'anonymous';
    }

    public render() {
        return (
            <Page title='Login' toolbar={{breadcrumbs: [{title: 'Login'}]}}>
                <div className='argo-container'>
                    {this.state.error && (
                        <p>
                            <i className='fa fa-exclamation-triangle' /> Failed to load user info {this.state.error.message}.
                        </p>
                    )}
                    <p>
                        <i className='fa fa-info-circle' /> You appear to be <b> logged {maybeLoggedIn() ? 'in as ' + this.username : 'out'}</b>. It may not be necessary to login
                        to use Argo, it depends on how it is configured.
                    </p>
                    <p>
                        Get your token using <code>argo auth token</code> and paste in this box.
                    </p>
                    <textarea id='token' cols={100} rows={20} defaultValue={getToken()} />
                    <div>
                        {maybeLoggedIn() && (
                            <button className='argo-button argo-button--base-o' onClick={() => logout()}>
                                <i className='fa fa-lock' /> Logout
                            </button>
                        )}
                        <button className='argo-button argo-button--base-o' onClick={() => login((document.getElementById('token') as HTMLInputElement).value)}>
                            <i className='fa fa-lock-open' /> Login
                        </button>
                    </div>
                </div>
            </Page>
        );
    }
}
