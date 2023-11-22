import {Layout, Notifications, NotificationsManager, NotificationType, Popup, PopupManager, PopupProps} from 'argo-ui';
import * as H from 'history';

import * as React from 'react';
import {useEffect, useState} from 'react';
import {Redirect, Route, Router, Switch} from 'react-router';
import {Version} from '../models';
import apidocs from './apidocs';
import clusterWorkflowTemplates from './cluster-workflow-templates';
import cronWorkflows from './cron-workflows';
import eventflow from './event-flow';
import eventSources from './event-sources';
import help from './help';
import login from './login';
import {ModalSwitch} from './modals/modal-switch';
import plugins from './plugins';
import reports from './reports';
import sensors from './sensors';
import {uiUrl} from './shared/base';
import {ChatButton} from './shared/components/chat-button';
import ErrorBoundary from './shared/components/error-boundary';
import {services} from './shared/services';
import {Utils} from './shared/utils';
import userinfo from './userinfo';
import {Widgets} from './widgets/widgets';
import workflowEventBindings from './workflow-event-bindings';
import workflowTemplates from './workflow-templates';
import workflows from './workflows';

const eventFlowUrl = uiUrl('event-flow');
const sensorUrl = uiUrl('sensors');
const workflowsUrl = uiUrl('workflows');
const workflowsEventBindingsUrl = uiUrl('workflow-event-bindings');
const workflowTemplatesUrl = uiUrl('workflow-templates');
const clusterWorkflowTemplatesUrl = uiUrl('cluster-workflow-templates');
const cronWorkflowsUrl = uiUrl('cron-workflows');
const eventSourceUrl = uiUrl('event-sources');
const pluginsUrl = uiUrl('plugins');
const helpUrl = uiUrl('help');
const apiDocsUrl = uiUrl('apidocs');
const userInfoUrl = uiUrl('userinfo');
const loginUrl = uiUrl('login');
const timelineUrl = uiUrl('timeline');
const reportsUrl = uiUrl('reports');

export function AppRouter({popupManager, history, notificationsManager}: {popupManager: PopupManager; history: H.History; notificationsManager: NotificationsManager}) {
    const [popupProps, setPopupProps] = useState<PopupProps>();
    const [modals, setModals] = useState<{string: boolean}>();
    const [version, setVersion] = useState<Version>();
    const [namespace, setNamespace] = useState<string>();
    const [navBarBackgroundColor, setNavBarBackgroundColor] = useState<string>();
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
        services.info.getUserInfo().then(userInfo => {
            Utils.userNamespace = userInfo.serviceAccountNamespace;
            setNamespace(Utils.currentNamespace);
        });
        services.info
            .getInfo()
            .then(info => {
                Utils.managedNamespace = info.managedNamespace;
                setNamespace(Utils.currentNamespace);
                setModals(info.modals);
                setNavBarBackgroundColor(info.navColor);
            })
            .then(() => services.info.getVersion())
            .then(setVersion)
            .catch(setError);
    }, []);

    const namespaceSuffix = Utils.managedNamespace ? '' : '/' + namespace;
    return (
        <>
            {popupProps && <Popup {...popupProps} />}
            <Router history={history}>
                <Switch>
                    <Route path={uiUrl('widgets')} component={Widgets} />
                    <Layout
                        navBarStyle={{backgroundColor: navBarBackgroundColor}}
                        navItems={[
                            {
                                title: 'Workflows',
                                path: workflowsUrl + namespaceSuffix,
                                iconClassName: 'fa fa-stream'
                            },
                            {
                                title: 'Workflow Templates',
                                path: workflowTemplatesUrl + namespaceSuffix,
                                iconClassName: 'fa fa-window-maximize'
                            },
                            {
                                title: 'Cluster Workflow Templates',
                                path: clusterWorkflowTemplatesUrl,
                                iconClassName: 'fa fa-window-restore'
                            },
                            {
                                title: 'Cron Workflows',
                                path: cronWorkflowsUrl + namespaceSuffix,
                                iconClassName: 'fa fa-clock'
                            },
                            {
                                title: 'Event Flow',
                                path: eventFlowUrl + namespaceSuffix,
                                iconClassName: 'fa fa-broadcast-tower'
                            },
                            {
                                title: 'Event Sources',
                                path: eventSourceUrl + namespaceSuffix,
                                iconClassName: 'fas fa-bolt'
                            },
                            {
                                title: 'Sensors',
                                path: sensorUrl + namespaceSuffix,
                                iconClassName: 'fa fa-satellite-dish'
                            },
                            {
                                title: 'Workflow Event Bindings',
                                path: workflowsEventBindingsUrl + namespaceSuffix,
                                iconClassName: 'fa fa-link'
                            },
                            {
                                title: 'Reports',
                                path: reportsUrl + namespaceSuffix,
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
                                title: 'Plugins',
                                path: pluginsUrl,
                                iconClassName: 'fa fa-puzzle-piece'
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
                                <Route path={eventFlowUrl} component={eventflow.component} />
                                <Route path={sensorUrl} component={sensors.component} />
                                <Route path={eventSourceUrl} component={eventSources.component} />
                                <Route path={workflowsUrl} component={workflows.component} />
                                <Route path={workflowsEventBindingsUrl} component={workflowEventBindings.component} />
                                <Route path={workflowTemplatesUrl} component={workflowTemplates.component} />
                                <Route path={clusterWorkflowTemplatesUrl} component={clusterWorkflowTemplates.component} />
                                <Route path={cronWorkflowsUrl} component={cronWorkflows.component} />
                                <Route path={reportsUrl} component={reports.component} />
                                <Route path={pluginsUrl} component={plugins.component} />
                                <Route exact={true} strict={true} path={helpUrl} component={help.component} />
                                <Route exact={true} strict={true} path={apiDocsUrl} component={apidocs.component} />
                                <Route exact={true} strict={true} path={userInfoUrl} component={userinfo.component} />
                                <Route exact={true} strict={true} path={loginUrl} component={login.component} />
                                {Utils.managedNamespace && <Redirect to={workflowsUrl} />}
                                {namespace && <Redirect to={workflowsUrl + '/' + namespace} />}
                            </Switch>
                        </ErrorBoundary>
                        <ChatButton />
                        {version && modals && <ModalSwitch version={version.version} modals={modals} />}
                    </Layout>
                </Switch>
            </Router>
        </>
    );
}
