import {createBrowserHistory} from 'history';
import * as PropTypes from 'prop-types';
import * as React from 'react';
import {Redirect, Route, RouteComponentProps, Router, Switch} from 'react-router';

import {Layout, NavigationManager, Notifications, NotificationsManager, Popup, PopupManager, PopupProps} from 'argo-ui';
import {uiUrl} from './shared/base';
import {ContextApis, Provider} from './shared/context';

import {Version} from '../models';
import apidocs from './apidocs';
import archivedWorkflows from './archived-workflows';
import clusterWorkflowTemplates from './cluster-workflow-templates';
import cronWorkflows from './cron-workflows';
import help from './help';
import login from './login';
import ErrorBoundary from './shared/components/error-boundary';
import {services} from './shared/services';
import {Utils} from './shared/utils';
import userinfo from './userinfo';
import workflowTemplates from './workflow-templates';
import workflows from './workflows';

const workflowsUrl = uiUrl('workflows');
const workflowTemplatesUrl = uiUrl('workflow-templates');
const clusterWorkflowTemplatesUrl = uiUrl('cluster-workflow-templates');

const cronWorkflowsUrl = uiUrl('cron-workflows');
const archivedWorkflowsUrl = uiUrl('archived-workflows');
const helpUrl = uiUrl('help');
const apiDocsUrl = uiUrl('apidocs');
const userInfoUrl = uiUrl('userinfo');
const loginUrl = uiUrl('login');
const timelineUrl = uiUrl('timeline');
const routes: {
    [path: string]: {component: React.ComponentType<RouteComponentProps<any>>};
} = {
    [workflowsUrl]: {component: workflows.component},
    [workflowTemplatesUrl]: {component: workflowTemplates.component},
    [clusterWorkflowTemplatesUrl]: {component: clusterWorkflowTemplates.component},
    [cronWorkflowsUrl]: {component: cronWorkflows.component},
    [archivedWorkflowsUrl]: {component: archivedWorkflows.component},
    [helpUrl]: {component: help.component},
    [apiDocsUrl]: {component: apidocs.component},
    [userInfoUrl]: {component: userinfo.component},
    [loginUrl]: {component: login.component}
};

export const history = createBrowserHistory();

export class App extends React.Component<{}, {version?: Version; popupProps: PopupProps; namespace?: string; error?: Error}> {
    public static childContextTypes = {
        history: PropTypes.object,
        apis: PropTypes.object
    };

    private popupManager: PopupManager;
    private notificationsManager: NotificationsManager;
    private navigationManager: NavigationManager;

    constructor(props: {}) {
        super(props);
        this.state = {popupProps: null};
        this.popupManager = new PopupManager();
        this.notificationsManager = new NotificationsManager();
        this.navigationManager = new NavigationManager(history);
    }

    public componentDidMount() {
        this.popupManager.popupProps.subscribe(popupProps => this.setState({popupProps}));
        services.info
            .getVersion()
            .then(version => this.setState({version}))
            .then(() => services.info.getInfo())
            .then(info => this.setState({namespace: Utils.getCurrentNamespace() || info.managedNamespace || ''}))
            .catch(error => this.setState({error}));
    }

    public render() {
        const providerContext: ContextApis = {
            notifications: this.notificationsManager,
            popup: this.popupManager,
            navigation: this.navigationManager,
            history
        };
        return (
            <Provider value={providerContext}>
                {this.state.popupProps && <Popup {...this.state.popupProps} />}
                <Router history={history}>
                    <Switch>
                        <Redirect exact={true} path={uiUrl('')} to={workflowsUrl} />
                        <Redirect from={timelineUrl} to={workflowsUrl} />
                        {this.state.namespace && false && (
                            <>
                                <Redirect exact={true} strict={true} from={workflowsUrl} to={workflowsUrl + '/' + this.state.namespace} />
                                <Redirect exact={true} strict={true} from={workflowTemplatesUrl} to={workflowTemplatesUrl + '/' + this.state.namespace} />
                                <Redirect exact={true} strict={true} from={clusterWorkflowTemplatesUrl} to={clusterWorkflowTemplatesUrl + '/?namespace=' + this.state.namespace} />
                                <Redirect exact={true} strict={true} from={cronWorkflowsUrl} to={cronWorkflowsUrl + '/' + this.state.namespace} />
                                <Redirect exact={true} strict={true} from={archivedWorkflowsUrl} to={archivedWorkflowsUrl + '/' + this.state.namespace} />
                            </>
                        )}
                        <ErrorBoundary>
                            <Layout
                                navItems={[
                                    {
                                        title: 'Timeline',
                                        path: workflowsUrl + '/' + this.state.namespace,
                                        iconClassName: 'fa fa-stream'
                                    },
                                    {
                                        title: 'Workflow Templates',
                                        path: workflowTemplatesUrl + '/' + this.state.namespace,
                                        iconClassName: 'fa fa-window-maximize'
                                    },
                                    {
                                        title: 'Cluster Workflow Templates',
                                        path: clusterWorkflowTemplatesUrl + '/?namespace=' + this.state.namespace,
                                        iconClassName: 'fa fa-window-restore'
                                    },
                                    {
                                        title: 'Cron Workflows',
                                        path: cronWorkflowsUrl + '/' + this.state.namespace,
                                        iconClassName: 'fa fa-clock'
                                    },
                                    {
                                        title: 'Archived Workflows',
                                        path: archivedWorkflowsUrl + '/' + this.state.namespace,
                                        iconClassName: 'fa fa-archive'
                                    },
                                    {title: 'User', path: userInfoUrl, iconClassName: 'fa fa-user-alt'},
                                    {title: 'API Docs', path: apiDocsUrl, iconClassName: 'fa fa-code'},
                                    {title: 'Help', path: helpUrl, iconClassName: 'fa fa-question-circle'}
                                ]}
                                version={() => <>{this.state.version ? this.state.version.version : 'unknown'}</>}>
                                <Notifications notifications={this.notificationsManager.notifications} />
                                {Object.keys(routes).map(path => {
                                    const route = routes[path];
                                    return <Route key={path} path={path} component={route.component} />;
                                })}
                            </Layout>
                        </ErrorBoundary>
                    </Switch>
                </Router>
            </Provider>
        );
    }

    public getChildContext() {
        return {
            history,
            apis: {
                popup: this.popupManager,
                notifications: this.notificationsManager
            }
        };
    }
}
