import {Page} from 'argo-ui';
import * as React from 'react';
import {useEffect, useState} from 'react';
import {GetUserInfoResponse} from '../../../models';
import {uiUrl} from '../../shared/base';
import {ErrorNotice} from '../../shared/components/error-notice';
import {Notice} from '../../shared/components/notice';
import {services} from '../../shared/services';
import {CliHelp} from './cli-help';

export function UserInfo() {
    const [error, setError] = useState<Error>(null);
    const [userInfo, setUserInfo] = useState<GetUserInfoResponse>();

    useEffect(() => {
        (async function getUserInfoWrapper() {
            try {
                const newUserInfo = await services.info.getUserInfo();
                setUserInfo(newUserInfo);
                setError(null);
            } catch (newError) {
                setError(newError);
            }
        })();
    }, []);

    return (
        <Page title='User Info' toolbar={{breadcrumbs: [{title: 'User Info'}]}}>
            <ErrorNotice error={error} />
            <Notice>
                <h3>
                    <i className='fa fa-user-alt' /> User Info
                </h3>
                {userInfo && (
                    <>
                        {userInfo.issuer && (
                            <dl>
                                <dt>Issuer:</dt>
                                <dd>{userInfo.issuer}</dd>
                            </dl>
                        )}
                        {userInfo.subject && (
                            <dl>
                                <dt>Subject:</dt>
                                <dd>{userInfo.subject}</dd>
                            </dl>
                        )}
                        {userInfo.groups && userInfo.groups.length > 0 && (
                            <dl>
                                <dt>Groups:</dt>
                                <dd>{userInfo.groups.join(', ')}</dd>
                            </dl>
                        )}
                        {userInfo.name && (
                            <dl>
                                <dt>Name:</dt>
                                <dd>{userInfo.name}</dd>
                            </dl>
                        )}
                        {userInfo.email && (
                            <dl>
                                <dt>Email:</dt>
                                <dd>{userInfo.email}</dd>
                            </dl>
                        )}
                        {userInfo.emailVerified && (
                            <dl>
                                <dt>Email Verified:</dt>
                                <dd>{userInfo.emailVerified}</dd>
                            </dl>
                        )}
                        {userInfo.serviceAccountName && (
                            <dl>
                                <dt>Service Account:</dt>
                                <dd>{userInfo.serviceAccountName}</dd>
                            </dl>
                        )}
                        {userInfo.serviceAccountNamespace && (
                            <dl>
                                <dt>Service Account Namespace:</dt>
                                <dd>{userInfo.serviceAccountNamespace}</dd>
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
