import {Page} from 'argo-ui';
import * as React from 'react';
import {RouteComponentProps} from 'react-router-dom';
import {GetUserInfoResponse} from '../../../models';
import {uiUrl} from '../../shared/base';
import {BasePage} from '../../shared/components/base-page';
import {ErrorNotice} from '../../shared/components/error-notice';
import {Notice} from '../../shared/components/notice';
import {services} from '../../shared/services';
import {CliHelp} from './cli-help';

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
                {<ErrorNotice error={this.state.error} />}
                <Notice>
                    <h3>
                        <i className='fa fa-user-alt' /> User Info
                    </h3>
                    {this.state.userInfo && (
                        <>
                            {this.state.userInfo.issuer && (
                                <dl>
                                    <dt>Issuer:</dt>
                                    <dd>{this.state.userInfo.issuer}</dd>
                                </dl>
                            )}
                            {this.state.userInfo.subject && (
                                <dl>
                                    <dt>Subject:</dt>
                                    <dd>{this.state.userInfo.subject}</dd>
                                </dl>
                            )}
                            {this.state.userInfo.groups && this.state.userInfo.groups.length > 0 && (
                                <dl>
                                    <dt>Groups:</dt>
                                    <dd>{this.state.userInfo.groups.join(', ')}</dd>
                                </dl>
                            )}
                            {this.state.userInfo.name && (
                                <dl>
                                    <dt>Name:</dt>
                                    <dd>{this.state.userInfo.name}</dd>
                                </dl>
                            )}
                            {this.state.userInfo.email && (
                                <dl>
                                    <dt>Email:</dt>
                                    <dd>{this.state.userInfo.email}</dd>
                                </dl>
                            )}
                            {this.state.userInfo.emailVerified && (
                                <dl>
                                    <dt>Email Verified:</dt>
                                    <dd>{this.state.userInfo.emailVerified}</dd>
                                </dl>
                            )}
                            {this.state.userInfo.serviceAccountName && (
                                <dl>
                                    <dt>Service Account:</dt>
                                    <dd>{this.state.userInfo.serviceAccountName}</dd>
                                </dl>
                            )}
                            {this.state.userInfo.serviceAccountNamespace && (
                                <dl>
                                    <dt>Service Account Namespace:</dt>
                                    <dd>{this.state.userInfo.serviceAccountNamespace}</dd>
                                </dl>
                            )}
                        </>
                    )}
                    <a className='argo-button argo-button--base-o' href={uiUrl('login')}>
                        <i className='fa fa-shield-alt' /> Login / Logout
                    </a>
                </Notice>
                <CliHelp />
            </Page>
        );
    }
}
