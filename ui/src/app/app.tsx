import {createBrowserHistory} from 'history';
import * as PropTypes from 'prop-types';
import * as React from 'react';
import {Redirect, Route, RouteComponentProps, Router, Switch} from 'react-router';

import {Layout, NavigationManager, Notifications, NotificationsManager, Popup, PopupManager, PopupProps} from 'argo-ui';
import {uiUrl} from './shared/base';
import {AppContext, ContextApis, Provider} from './shared/context';

import help from './help';
import login from './login';
import workflows from './workflows';

const workflowsUrl = uiUrl('workflows');
const helpUrl = uiUrl('help');
const loginUrl = uiUrl('login');
const timelineUrl = uiUrl('timeline');
const routes: {
    [path: string]: {component: React.ComponentType<RouteComponentProps<any>>};
} = {
    [workflowsUrl]: {component: workflows.component},
    [helpUrl]: {component: help.component},
    [loginUrl]: {component: login.component}
};

const bases = document.getElementsByTagName('base');
const base = bases.length > 0 ? bases[0].getAttribute('href') || '/' : '/';
export const history = createBrowserHistory({basename: base});

const navItems = [
    {
        title: 'Timeline',
        path: workflowsUrl,
        iconClassName: 'argo-icon-timeline'
    },
    {
        title: 'Help',
        path: helpUrl,
        iconClassName: 'argo-icon-docs'
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
                        <Route
                            path={timelineUrl}
                            component={
                                class ToWorkflows extends React.Component {
                                    public static contextTypes = {router: PropTypes.object};
                                    public render() {
                                        return <div />;
                                    }
                                    public componentWillMount() {
                                        const router = (this.context as AppContext).router;
                                        router.history.push(router.route.location.pathname.replace(timelineUrl, workflowsUrl));
                                    }
                                }
                            }
                        />
                        <Layout navItems={navItems}>
                            <Notifications notifications={this.notificationsManager.notifications} />
                            {Object.keys(routes).map(path => {
                                const route = routes[path];
                                return <Route key={path} path={path} component={route.component} />;
                            })}
                        </Layout>
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
