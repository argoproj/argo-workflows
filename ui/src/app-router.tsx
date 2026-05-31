import {Layout} from 'argo-ui/src/components/layout/layout';
import {NotificationsManager} from 'argo-ui/src/components/notifications/notification-manager';
import {Notifications, NotificationType} from 'argo-ui/src/components/notifications/notifications';
import {Popup, PopupProps} from 'argo-ui/src/components/popup/popup';
import {PopupManager} from 'argo-ui/src/components/popup/popup-manager';
import * as H from 'history';
import * as React from 'react';
import {useEffect, useState} from 'react';
import {unstable_HistoryRouter as HistoryRouter, Navigate, Route, Routes} from 'react-router-dom';

import apiDocs from './api-docs';
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
import {Version} from './shared/models';
import * as nsUtils from './shared/namespaces';
import {services} from './shared/services';
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
    nsUtils.setOnNamespaceChange(setNamespace);
    useEffect(() => {
        const sub = popupManager.popupProps.subscribe(setPopupProps);
        return () => sub.unsubscribe();
    }, [popupManager]);
    useEffect(() => {
        services.info.getUserInfo().then(userInfo => {
            nsUtils.setUserNamespace(userInfo.serviceAccountNamespace);
            setNamespace(nsUtils.getCurrentNamespace());
        });
        services.info
            .getInfo()
            .then(info => {
                nsUtils.setManagedNamespace(info.managedNamespace);
                setNamespace(nsUtils.getCurrentNamespace());
                setModals(info.modals);
                setNavBarBackgroundColor(info.navColor);
            })
            .then(() => services.info.getVersion())
            .then(setVersion)
            .catch(setError);
    }, []);

    const managedNamespace = nsUtils.getManagedNamespace();
    const namespaceSuffix = managedNamespace ? '' : '/' + (namespace || '');

    // The default route when no other route matches: redirect to the workflows list (optionally scoped to the namespace).
    const fallbackUrl = managedNamespace ? workflowsUrl : namespace ? workflowsUrl + '/' + namespace : workflowsUrl;

    return (
        <>
            {popupProps && <Popup {...popupProps} />}
            <HistoryRouter history={history as any}>
                <Routes>
                    <Route path={loginUrl} element={<login.component />} />
                    <Route path={uiUrl('widgets') + '/*'} element={<Widgets />} />
                    <Route
                        path='*'
                        element={
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
                                        title: 'User Info',
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
                                    <Routes>
                                        <Route path={timelineUrl} element={<Navigate to={workflowsUrl} replace />} />
                                        <Route path={eventFlowUrl + '/*'} element={<eventflow.component />} />
                                        <Route path={sensorUrl + '/*'} element={<sensors.component />} />
                                        <Route path={eventSourceUrl + '/*'} element={<eventSources.component />} />
                                        <Route path={workflowsUrl + '/*'} element={<workflows.component />} />
                                        <Route path={workflowsEventBindingsUrl + '/*'} element={<workflowEventBindings.component />} />
                                        <Route path={workflowTemplatesUrl + '/*'} element={<workflowTemplates.component />} />
                                        <Route path={clusterWorkflowTemplatesUrl + '/*'} element={<clusterWorkflowTemplates.component />} />
                                        <Route path={cronWorkflowsUrl + '/*'} element={<cronWorkflows.component />} />
                                        <Route path={reportsUrl + '/*'} element={<reports.component />} />
                                        <Route path={pluginsUrl + '/*'} element={<plugins.component />} />
                                        <Route path={helpUrl} element={<help.component />} />
                                        <Route path={apiDocsUrl} element={<apiDocs.component />} />
                                        <Route path={userInfoUrl} element={<userinfo.component />} />
                                        <Route path='*' element={<Navigate to={fallbackUrl} replace />} />
                                    </Routes>
                                </ErrorBoundary>
                                <ChatButton />
                                {version && modals && <ModalSwitch version={version.version} modals={modals} />}
                            </Layout>
                        }
                    />
                </Routes>
            </HistoryRouter>
        </>
    );
}
