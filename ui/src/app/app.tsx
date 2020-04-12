import {createBrowserHistory} from 'history';
import * as PropTypes from 'prop-types';
import * as React from 'react';
import {Redirect, Route, RouteComponentProps, Router, Switch} from 'react-router';

import {Layout, NavigationManager, Notifications, NotificationsManager, Popup, PopupManager, PopupProps} from 'argo-ui';
import {uiUrl} from './shared/base';
import {ContextApis, Provider} from './shared/context';

import archivedWorkflows from './archived-workflows';
import clusterWorkflowTemplates from './cluster-workflow-templates';
import cronWorkflows from './cron-workflows';
import help from './help';
import login from './login';
import ErrorBoundary from './shared/components/error-boundary';
import workflowTemplates from './workflow-templates';
import workflows from './workflows';

const workflowsUrl = uiUrl('workflows');
const workflowTemplatesUrl = uiUrl('workflow-templates');
const clusterWorkflowTemplatesUrl = uiUrl('cluster-workflow-templates');

const cronWorkflowUrl = uiUrl('cron-workflows');
const archivedWorkflowUrl = uiUrl('archived-workflows');
const helpUrl = uiUrl('help');
const loginUrl = uiUrl('login');
const timelineUrl = uiUrl('timeline');
const routes: {
    [path: string]: {component: React.ComponentType<RouteComponentProps<any>>};
} = {
    [workflowsUrl]: {component: workflows.component},
    [workflowTemplatesUrl]: {component: workflowTemplates.component},
    [clusterWorkflowTemplatesUrl]: {component: clusterWorkflowTemplates.component},
    [cronWorkflowUrl]: {component: cronWorkflows.component},
    [archivedWorkflowUrl]: {component: archivedWorkflows.component},
    [helpUrl]: {component: help.component},
    [loginUrl]: {component: login.component}
};

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
        path: cronWorkflowUrl,
        iconClassName: 'fa fa-clock'
    },
    {
        title: 'Archived Workflows',
        path: archivedWorkflowUrl,
        iconClassName: 'fa fa-archive'
    },
    {
        title: 'Login',
        path: loginUrl,
        iconClassName: 'fa fa-user-circle'
    },
    {
        title: 'Help',
        path: helpUrl,
        iconClassName: 'fa fa-question-circle'
    }
];

export class App extends React.Component<{}, {popupProps: PopupProps}> {
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
                        <Redirect from={timelineUrl} to={uiUrl('workflows')} />
                        <ErrorBoundary>
                            <Layout navItems={navItems}>
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
