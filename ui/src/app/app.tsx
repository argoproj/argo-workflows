import {NavigationManager, NotificationsManager, PopupManager} from 'argo-ui';

import {createBrowserHistory} from 'history';
import * as React from 'react';
import {AppRouter} from './app-router';
import {ContextApis, Provider} from './shared/context';

const history = createBrowserHistory();

export function App() {
    const popupManager: PopupManager = new PopupManager();
    const notificationsManager: NotificationsManager = new NotificationsManager();
    const navigationManager: NavigationManager = new NavigationManager(history);

    const providerContext: ContextApis = {
        notifications: notificationsManager,
        popup: popupManager,
        navigation: navigationManager,
        history
    };

    return (
        <Provider value={providerContext}>
            <AppRouter history={history} notificationsManager={notificationsManager} popupManager={popupManager} />
        </Provider>
    );
}
