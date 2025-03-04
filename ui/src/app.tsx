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
    const [error, setError] = useState<Error | null>(null);
    const popupManager: PopupManager = new PopupManager();
    const notificationsManager: NotificationsManager = new NotificationsManager();
    const navigationManager: NavigationManager = new NavigationManager(history);

    const providerContext: ContextApis = {
        notifications: notificationsManager,
        popup: popupManager,
        navigation: navigationManager,
        history
    };

    // Add a small delay to ensure all resources are properly loaded
    useEffect(() => {
        const timer = setTimeout(() => {
            try {
                // Any initialization logic that might fail can go here
                setIsLoading(false);
            } catch (err) {
                setError(err instanceof Error ? err : new Error('An unexpected error occurred'));
                setIsLoading(false);
            }
        }, 300);

        return () => clearTimeout(timer);
    }, []);

    // Global error boundary
    useEffect(() => {
        const handleError = (event: ErrorEvent) => {
            event.preventDefault();
            setError(event.error || new Error(event.message));
        };

        window.addEventListener('error', handleError);

        return () => {
            window.removeEventListener('error', handleError);
        };
    }, []);

    // Handle any errors during initialization
    if (error) {
        return (
            <div className='error-container'>
                <h1>Application Error</h1>
                <p>An error occurred while loading the application. Please try refreshing the page.</p>
                <pre>{error.message}</pre>
                <button onClick={() => window.location.reload()}>Refresh Page</button>
            </div>
        );
    }

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
