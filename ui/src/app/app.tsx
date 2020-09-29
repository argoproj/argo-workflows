import {createBrowserHistory} from 'history';
import * as PropTypes from 'prop-types';
import * as React from 'react';
import {Redirect, Route, Router, Switch} from 'react-router';

import {Layout, NavigationManager, Notifications, NotificationsManager, Popup, PopupManager, PopupProps} from 'argo-ui';
import {uiUrl} from './shared/base';
import {ContextApis, Provider} from './shared/context';

import {NotificationType} from 'argo-ui/src/index';
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

export const history = createBrowserHistory();

const navItems = [
    {
        title: 'Timeline',
        path: workflowsUrl,
        iconClassName: 'fa fa-stream'
    },
    {
        title: 'Workflow Templates',
        path: workflowTemplatesUrl,
        iconClassName: 'fa fa-window-maximize'
    },
    {
        title: 'Cluster Workflow Templates',
        path: clusterWorkflowTemplatesUrl,
        iconClassName: 'fa fa-window-restore'
    },
    {
        title: 'Cron Workflows',
        path: cronWorkflowsUrl,
        iconClassName: 'fa fa-clock'
    },
    {
        title: 'Archived Workflows',
        path: archivedWorkflowsUrl,
        iconClassName: 'fa fa-archive'
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
];

export class App extends React.Component<{}, {version?: Version; popupProps: PopupProps; namespace?: string}> {
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
        Utils.onNamespaceChange = namespace => {
            this.setState({namespace});
        };
    }

    public componentDidMount() {
        this.popupManager.popupProps.subscribe(popupProps => this.setState({popupProps}));
        services.info
            .getVersion()
            .then(version => this.setState({version}))
            .then(() => services.info.getInfo())
            .then(info => this.setState({namespace: info.managedNamespace || Utils.getCurrentNamespace() || ''}))
            .catch(error => {
                this.notificationsManager.show({
                    content: 'Failed to load ' + error,
                    type: NotificationType.Error
                });
            });
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
                    <Layout navItems={navItems} version={() => <>{this.state.version ? this.state.version.version : 'unknown'}</>}>
                        <Notifications notifications={this.notificationsManager.notifications} />
                        <ErrorBoundary>
                            <Switch>
                                <Route exact={true} strict={true} path={uiUrl('')}>
                                    <Redirect to={workflowsUrl} />
                                </Route>
                                <Route exact={true} strict={true} path={timelineUrl}>
                                    <Redirect to={workflowsUrl} />
                                </Route>
                                {this.state.namespace && (
                                    <Route exact={true} strict={true} path={workflowsUrl}>
                                        <Redirect to={this.workflowsUrl} />
                                    </Route>
                                )}
                                {this.state.namespace && (
                                    <Route exact={true} strict={true} path={workflowTemplatesUrl}>
                                        <Redirect to={this.workflowTemplatesUrl} />
                                    </Route>
                                )}
                                {this.state.namespace && (
                                    <Route exact={true} strict={true} path={cronWorkflowsUrl}>
                                        <Redirect to={this.cronWorkflowsUrl} />
                                    </Route>
                                )}
                                {this.state.namespace && (
                                    <Route exact={true} strict={true} path={archivedWorkflowsUrl}>
                                        <Redirect to={this.archivedWorkflowsUrl} />
                                    </Route>
                                )}
                                <Route path={workflowsUrl} component={workflows.component} />
                                <Route path={workflowTemplatesUrl} component={workflowTemplates.component} />
                                <Route path={clusterWorkflowTemplatesUrl} component={clusterWorkflowTemplates.component} />
                                <Route path={cronWorkflowsUrl} component={cronWorkflows.component} />
                                <Route path={archivedWorkflowsUrl} component={archivedWorkflows.component} />
                                <Route exact={true} strict={true} path={helpUrl} component={help.component} />
                                <Route exact={true} strict={true} path={apiDocsUrl} component={apidocs.component} />
                                <Route exact={true} strict={true} path={userInfoUrl} component={userinfo.component} />
                                <Route exact={true} strict={true} path={loginUrl} component={login.component} />
                            </Switch>
                        </ErrorBoundary>
                    </Layout>
                </Router>
            </Provider>
        );
    }

    private get archivedWorkflowsUrl() {
        return archivedWorkflowsUrl + '/' + (this.state.namespace || '');
    }

    private get cronWorkflowsUrl() {
        return cronWorkflowsUrl + '/' + (this.state.namespace || '');
    }

    private get workflowTemplatesUrl() {
        return workflowTemplatesUrl + '/' + (this.state.namespace || '');
    }

    private get workflowsUrl() {
        return workflowsUrl + '/' + (this.state.namespace || '');
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
