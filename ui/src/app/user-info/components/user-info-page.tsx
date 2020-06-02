import {Page} from 'argo-ui';
import * as React from 'react';
import {RouteComponentProps} from 'react-router-dom';
import {uiUrl} from '../../shared/base';
import {BasePage} from '../../shared/components/base-page';

require('./user-info.scss');

export class UserInfoPage extends BasePage<RouteComponentProps<void>, {}> {
    public render() {
        return (
            <Page title='User Info' toolbar={{breadcrumbs: [{title: 'User'}]}}>
                <div className='argo-container'>
                    <div className='white-box'>
                        <h3>
                            <i className='fa fa-user-circle' /> User Info
                        </h3>
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
