import {Page} from 'argo-ui';
import * as React from 'react';
import {RouteComponentProps} from 'react-router-dom';
import {GetUserInfoResponse} from '../../../models';
import {uiUrl} from '../../shared/base';
import {BasePage} from '../../shared/components/base-page';
import {ErrorNotice} from '../../shared/components/error-notice';
import {services} from '../../shared/services';

interface State {
    error?: Error;
    userInfo?: GetUserInfoResponse;
}

export class UserInfo extends BasePage<RouteComponentProps<any>, State> {
    constructor(props: RouteComponentProps<any>, context: any) {
        super(props, context);
        this.state = {};
    }

    public componentDidMount() {
        services.info
            .getUserInfo()
            .then(userInfo => this.setState({error: null, userInfo}))
            .catch(error => this.setState({error}));
    }

    public render() {
        return (
            <Page title='User Info' toolbar={{breadcrumbs: [{title: 'User Info'}]}}>
                <div className='argo-container'>
                    {this.state.error && <ErrorNotice error={this.state.error} />}
                    <div className='white-box'>
                        <h3>
                            <i className='fa fa-user-alt' /> User Info
                        </h3>
                        {this.state.userInfo && (
                            <>
                                <p>Issuer: {this.state.userInfo.issuer || '-'}</p>
                                <p>Subject: {this.state.userInfo.subject || '-'}</p>
                                <p>Groups: {(this.state.userInfo.groups && this.state.userInfo.groups.length > 0 && this.state.userInfo.groups.join(', ')) || '-'}</p>
                            </>
                        )}
                        <a className='argo-button argo-button--base-o' href={uiUrl('login')}>
                            <i className='fa fa-shield-alt' /> Login / Logout
                        </a>
                    </div>
                </div>
            </Page>
        );
    }
}
