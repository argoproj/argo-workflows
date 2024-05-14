import {createBrowserHistory} from 'history';
import * as React from 'react';
import {NavigationManager} from 'argo-ui/src/components/navigation';
import {NotificationsManager} from 'argo-ui/src/components/notifications/notification-manager';
import {PopupManager} from 'argo-ui/src/components/popup/popup-manager';

import 'argo-ui/src/styles/main.scss';

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
