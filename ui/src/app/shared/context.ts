import type {NavigationApi} from 'argo-ui/src/components/navigation';
import type {NotificationsApi} from 'argo-ui/src/components/notifications/notification-manager';
import type {PopupApi} from 'argo-ui/src/components/popup/popup-manager';
import type {AppContext as ArgoAppContext} from 'argo-ui/src/context';
import {History} from 'history';
import * as React from 'react';

export type AppContext = ArgoAppContext & {apis: {popup: PopupApi; notifications: NotificationsApi; navigation: NavigationApi; baseHref: string}};

export interface ContextApis {
    popup: PopupApi;
    notifications: NotificationsApi;
    navigation: NavigationApi;
    history: History;
}

export const Context = React.createContext<ContextApis>(null);
export const {Provider, Consumer} = Context;
