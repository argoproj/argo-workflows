import {Layout, Notifications, NotificationsManager, NotificationType, Popup, PopupManager, PopupProps} from 'argo-ui';
import * as H from 'history';

import * as React from 'react';
import {useEffect, useState} from 'react';
import {Redirect, Route, Router, Switch} from 'react-router';
import {Version} from '../models';
import apidocs from './apidocs';
import archivedWorkflows from './archived-workflows';
import clusterWorkflowTemplates from './cluster-workflow-templates';
import cronWorkflows from './cron-workflows';
import events from './events';
import help from './help';
import login from './login';
import reports from './reports';
import {uiUrl} from './shared/base';
import {ChatButton} from './shared/components/chat-button';
import ErrorBoundary from './shared/components/error-boundary';
import {services} from './shared/services';
import {Utils} from './shared/utils';
import userinfo from './userinfo';
import workflowEventBindings from './workflow-event-bindings';
import workflowTemplates from './workflow-templates';
import workflows from './workflows';

const eventsUrl = uiUrl('events');
const workflowsUrl = uiUrl('workflows');
const workflowsEventBindingsUrl = uiUrl('workflow-event-bindings');
const workflowTemplatesUrl = uiUrl('workflow-templates');
const clusterWorkflowTemplatesUrl = uiUrl('cluster-workflow-templates');
const cronWorkflowsUrl = uiUrl('cron-workflows');
const archivedWorkflowsUrl = uiUrl('archived-workflows');
const helpUrl = uiUrl('help');
const apiDocsUrl = uiUrl('apidocs');
const userInfoUrl = uiUrl('userinfo');
const loginUrl = uiUrl('login');
const timelineUrl = uiUrl('timeline');
const reportsUrl = uiUrl('reports');

export const AppRouter = ({popupManager, history, notificationsManager}: {popupManager: PopupManager; history: H.History; notificationsManager: NotificationsManager}) => {
    const [popupProps, setPopupProps] = useState<PopupProps>();
    const [version, setVersion] = useState<Version>();
    const [namespace, setNamespace] = useState<string>();
    const setError = (error: Error) => {
        notificationsManager.show({
            content: 'Failed to load version/info ' + error,
            type: NotificationType.Error
        });
    };
    Utils.onNamespaceChange = setNamespace;
    useEffect(() => {
        const sub = popupManager.popupProps.subscribe(setPopupProps);
        return () => sub.unsubscribe();
    }, [popupManager]);
    useEffect(() => {
        services.info
            .getInfo()
            .then(info => setNamespace(info.managedNamespace || Utils.getCurrentNamespace() || ''))
            .then(() => services.info.getVersion())
            .then(setVersion)
            .catch(setError);
    }, []);

    return (
        <>
            {popupProps && <Popup {...popupProps} />}
            <Router history={history}>
                <Layout
                    navItems={[
                        {
                            title: 'Events',
                            path: eventsUrl + '/' + namespace,
                            iconClassName: 'fa fa-broadcast-tower'
                        },
                        {
                            title: 'Workflows',
                            path: workflowsUrl + '/' + namespace,
                            iconClassName: 'fa fa-stream'
                        },
                        {
                            title: 'Workflow Templates',
                            path: workflowTemplatesUrl + '/' + namespace,
                            iconClassName: 'fa fa-window-maximize'
                        },
                        {
                            title: 'Cluster Workflow Templates',
                            path: clusterWorkflowTemplatesUrl,
                            iconClassName: 'fa fa-window-restore'
                        },
                        {
                            title: 'Cron Workflows',
                            path: cronWorkflowsUrl + '/' + namespace,
                            iconClassName: 'fa fa-clock'
                        },
                        {
                            title: 'Workflow Event Bindings',
                            path: workflowsEventBindingsUrl + '/' + namespace,
                            iconClassName: 'fa fa-cloud'
                        },
                        {
                            title: 'Archived Workflows',
                            path: archivedWorkflowsUrl + '/' + namespace,
                            iconClassName: 'fa fa-archive'
                        },
                        {
                            title: 'Reports',
                            path: reportsUrl + '/' + namespace,
                            iconClassName: 'fa fa-chart-bar'
                        },
                        {
                            title: 'User',
                            path: userInfoUrl,
                            iconClassName: 'fa fa-user-alt'
                        },
                        {
                            title: 'API Docs',
                            path: apiDocsUrl,
                            iconClassName: 'fa fa-code'
                        },
                        {
                            title: 'Help',
                            path: helpUrl,
                            iconClassName: 'fa fa-question-circle'
                        }
                    ]}
                    version={() => <>{version ? version.version : 'unknown'}</>}>
                    <Notifications notifications={notificationsManager.notifications} />
                    <ErrorBoundary>
                        <Switch>
                            <Route exact={true} strict={true} path={timelineUrl}>
                                <Redirect to={workflowsUrl} />
                            </Route>
                            <Route path={eventsUrl} component={events.component} />
                            <Route path={workflowsUrl} component={workflows.component} />
                            <Route path={workflowsEventBindingsUrl} component={workflowEventBindings.component} />
                            <Route path={workflowTemplatesUrl} component={workflowTemplates.component} />
                            <Route path={clusterWorkflowTemplatesUrl} component={clusterWorkflowTemplates.component} />
                            <Route path={cronWorkflowsUrl} component={cronWorkflows.component} />
                            <Route path={archivedWorkflowsUrl} component={archivedWorkflows.component} />
                            <Route path={reportsUrl} component={reports.component} />
                            <Route exact={true} strict={true} path={helpUrl} component={help.component} />
                            <Route exact={true} strict={true} path={apiDocsUrl} component={apidocs.component} />
                            <Route exact={true} strict={true} path={userInfoUrl} component={userinfo.component} />
                            <Route exact={true} strict={true} path={loginUrl} component={login.component} />
                            {namespace && <Redirect to={workflowsUrl + '/' + namespace} />}
                        </Switch>
                    </ErrorBoundary>
                    <ChatButton />
                </Layout>
            </Router>
        </>
    );
};
