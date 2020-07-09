import {Page} from 'argo-ui';
import * as React from 'react';
import {RouteComponentProps} from 'react-router-dom';
import {WhoAmIResponse} from '../../../models';
import {uiUrl} from '../../shared/base';
import {BasePage} from '../../shared/components/base-page';
import {services} from '../../shared/services';

interface State {
    error?: Error;
    whoAmI?: WhoAmIResponse;
}

export class User extends BasePage<RouteComponentProps<any>, State> {
    constructor(props: RouteComponentProps<any>, context: any) {
        super(props, context);
        this.state = {};
    }

    public componentDidMount() {
        services.info
            .whoAmI()
            .then(whoAmI => this.setState({whoAmI}))
            .catch(error => this.setState({error}));
    }

    public render() {
        if (this.state.error) {
            throw this.state.error;
        }
        return (
            <Page title='User' toolbar={{breadcrumbs: [{title: 'User'}]}}>
                <div className='argo-container'>
                    <div className='white-box'>
                        <h3>
                            <i className='fa fa-user-alt' /> User
                        </h3>
                        <p>{this.state.whoAmI && this.state.whoAmI.subject}</p>
                        <a className='argo-button argo-button--base-o' href={uiUrl('login')}>
                            <i className='fa fa-shield-alt' /> Login / Logout
                        </a>
                    </div>
                </div>
            </Page>
        );
    }
}
