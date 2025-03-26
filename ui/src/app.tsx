import {NavigationManager} from 'argo-ui/src/components/navigation';
import {NotificationsManager} from 'argo-ui/src/components/notifications/notification-manager';
import {PopupManager} from 'argo-ui/src/components/popup/popup-manager';
import {createBrowserHistory} from 'history';
import * as React from 'react';
import {useEffect, useState} from 'react';

import 'argo-ui/src/styles/main.scss';
import './app.scss';

import {AppRouter} from './app-router';
import {ContextApis, Provider} from './shared/context';

const history = createBrowserHistory();

export function App() {
    const [isLoading, setIsLoading] = useState(true);
    const popupManager: PopupManager = new PopupManager();
    const notificationsManager: NotificationsManager = new NotificationsManager();
    const navigationManager: NavigationManager = new NavigationManager(history);

    const providerContext: ContextApis = {
        notifications: notificationsManager,
        popup: popupManager,
        navigation: navigationManager,
        history
    };

    // Show loading indicator briefly to ensure UI is ready before rendering
    useEffect(() => {
        // Short delay to ensure the UI is ready to render
        // This helps prevent the white screen issue by giving the app time to initialize
        const timer = setTimeout(() => {
            setIsLoading(false);
        }, 100);

        return () => clearTimeout(timer);
    }, []);

    // Show loading indicator while initializing
    if (isLoading) {
        return (
            <div className='loading-container'>
                <div style={{textAlign: 'center'}}>
                    <div className='loading-spinner'></div>
                    <p>Loading Argo Workflows...</p>
                </div>
            </div>
        );
    }

    return (
        <Provider value={providerContext}>
            <AppRouter history={history} notificationsManager={notificationsManager} popupManager={popupManager} />
        </Provider>
    );
}
