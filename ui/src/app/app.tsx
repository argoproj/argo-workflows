import {NavigationManager, NotificationsManager, PopupManager} from 'argo-ui';

import {createBrowserHistory} from 'history';
import * as PropTypes from 'prop-types';
import * as React from 'react';
import {AppRouter} from './app-router';
import {ContextApis, Provider} from './shared/context';
import {services} from './shared/services';

const history = createBrowserHistory();

export class App extends React.Component<{}> {
    public static childContextTypes = {
        history: PropTypes.object,
        apis: PropTypes.object
    };

    private readonly popupManager: PopupManager;
    private readonly notificationsManager: NotificationsManager;
    private readonly navigationManager: NavigationManager;

    constructor(props: {}) {
        super(props);
        this.popupManager = new PopupManager();
        this.notificationsManager = new NotificationsManager();
        this.navigationManager = new NavigationManager(history);
    }

    public async componentDidMount() {
        const settings = await services.info.getSettings();
        if (settings.uiCssURL) {
            const link = document.createElement('link');
            link.href = settings.uiCssURL;
            link.rel = 'stylesheet';
            link.type = 'text/css';
            document.head.appendChild(link);
        }
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
                <AppRouter history={history} notificationsManager={this.notificationsManager} popupManager={this.popupManager} />
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
