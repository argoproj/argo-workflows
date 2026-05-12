import {Layout} from 'argo-ui/src/components/layout/layout';
import {NotificationsManager} from 'argo-ui/src/components/notifications/notification-manager';
import {Notifications, NotificationType} from 'argo-ui/src/components/notifications/notifications';
import {Popup, PopupProps} from 'argo-ui/src/components/popup/popup';
import {PopupManager} from 'argo-ui/src/components/popup/popup-manager';
import * as H from 'history';
import * as React from 'react';
import {useEffect, useState} from 'react';
import {Redirect, Route, Router, Switch} from 'react-router';

import {ModalSwitch} from './modals/modal-switch';
import {uiUrl} from './shared/base';
import {ChatButton} from './shared/components/chat-button';
import ErrorBoundary from './shared/components/error-boundary';
import {Version} from './shared/models';
import * as nsUtils from './shared/namespaces';
import {services} from './shared/services';
import {lazyImport} from './shared/utils/lazy-import';
import {Widgets} from './widgets/widgets';

// Route-level code splitting with retry logic.
// Each module exposes a `.component` property, so we re-export it as the
// default so React.lazy (which requires a default export) can consume it.
const ApiDocs = React.lazy(() => lazyImport(() => import('./api-docs').then(m => ({default: m.default.component}))));
const ClusterWorkflowTemplates = React.lazy(() => lazyImport(() => import('./cluster-workflow-templates').then(m => ({default: m.default.component}))));
const CronWorkflows = React.lazy(() => lazyImport(() => import('./cron-workflows').then(m => ({default: m.default.component}))));
const EventFlow = React.lazy(() => lazyImport(() => import('./event-flow').then(m => ({default: m.default.component}))));
const EventSources = React.lazy(() => lazyImport(() => import('./event-sources').then(m => ({default: m.default.component}))));
const Help = React.lazy(() => lazyImport(() => import('./help').then(m => ({default: m.default.component}))));
const Login = React.lazy(() => lazyImport(() => import('./login').then(m => ({default: m.default.component}))));
const Plugins = React.lazy(() => lazyImport(() => import('./plugins').then(m => ({default: m.default.component}))));
const Reports = React.lazy(() => lazyImport(() => import('./reports').then(m => ({default: m.default.component}))));
const Sensors = React.lazy(() => lazyImport(() => import('./sensors').then(m => ({default: m.default.component}))));
const UserInfo = React.lazy(() => lazyImport(() => import('./userinfo').then(m => ({default: m.default.component}))));
const WorkflowEventBindings = React.lazy(() => lazyImport(() => import('./workflow-event-bindings').then(m => ({default: m.default.component}))));
const WorkflowTemplates = React.lazy(() => lazyImport(() => import('./workflow-templates').then(m => ({default: m.default.component}))));
const Workflows = React.lazy(() => lazyImport(() => import('./workflows').then(m => ({default: m.default.component}))));

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
    return (
        <>
            {popupProps && <Popup {...popupProps} />}
            <Router history={history}>
                <Switch>
                    <Route exact={true} strict={true} path={loginUrl} component={Login} />
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
                            <React.Suspense fallback={<div>Loading...</div>}>
                                <Switch>
                                    <Route exact={true} strict={true} path={timelineUrl}>
                                        <Redirect to={workflowsUrl} />
                                    </Route>
                                    <Route path={eventFlowUrl} component={EventFlow} />
                                    <Route path={sensorUrl} component={Sensors} />
                                    <Route path={eventSourceUrl} component={EventSources} />
                                    <Route path={workflowsUrl} component={Workflows} />
                                    <Route path={workflowsEventBindingsUrl} component={WorkflowEventBindings} />
                                    <Route path={workflowTemplatesUrl} component={WorkflowTemplates} />
                                    <Route path={clusterWorkflowTemplatesUrl} component={ClusterWorkflowTemplates} />
                                    <Route path={cronWorkflowsUrl} component={CronWorkflows} />
                                    <Route path={reportsUrl} component={Reports} />
                                    <Route path={pluginsUrl} component={Plugins} />
                                    <Route exact={true} strict={true} path={helpUrl} component={Help} />
                                    <Route exact={true} strict={true} path={apiDocsUrl} component={ApiDocs} />
                                    <Route exact={true} strict={true} path={userInfoUrl} component={UserInfo} />
                                    {managedNamespace && <Redirect to={workflowsUrl} />}
                                    {namespace && <Redirect to={workflowsUrl + '/' + namespace} />}
                                </Switch>
                            </React.Suspense>
                        </ErrorBoundary>
                        <ChatButton />
                        {version && modals && <ModalSwitch version={version.version} modals={modals} />}
                    </Layout>
                </Switch>
            </Router>
        </>
    );
}
