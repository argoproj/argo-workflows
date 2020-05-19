import {Page} from 'argo-ui';
import * as React from 'react';
import {RouteComponentProps} from 'react-router-dom';
import {User} from '../../../models';
import {uiUrl} from '../../shared/base';
import {BasePage} from '../../shared/components/base-page';
import {services} from '../../shared/services';

require('./user-info.scss');

interface State {
    user?: User;
    error?: Error;
}

export class UserInfoPage extends BasePage<RouteComponentProps<void>, State> {
    constructor(props: RouteComponentProps<void>, context: any) {
        super(props, context);
        this.state = {};
    }

    public componentDidMount() {
        services.info
            .getUser()
            .then(user => this.setState({user}))
            .catch(error => this.setState({error}));
    }

    public render() {
        return (
            <Page title='User Info' toolbar={{breadcrumbs: [{title: 'User'}]}}>
                <div className='argo-container'>
                    <div className='white-box'>
                        <h3>
                            <i className='fa fa-user-circle' /> User Info
                        </h3>
                        {this.state.error && (
                            <p>
                                <i className='fa fa-exclamation-triangle status-icon--failed' /> Failed to load user: {this.state.error.message}
                            </p>
                        )}
                        {this.state.user && (
                            <>
                                <p>Username: {this.state.user.name || '-'}</p>
                                {this.state.user.groups ? (
                                    <>
                                        <p>Groups:</p>
                                        <ul>
                                            {this.state.user.groups.map(group => (
                                                <li>{group}</li>
                                            ))}
                                        </ul>
                                    </>
                                ) : (
                                    <p>No groups</p>
                                )}
                            </>
                        )}
                        <p>
                            <button className='argo-button argo-button--base-o' onClick={() => (document.location.href = uiUrl('login'))}>
                                <i className='fa fa-sign-in-alt' /> Login/Logout
                            </button>
                        </p>
                    </div>
                </div>
            </Page>
        );
    }
}
