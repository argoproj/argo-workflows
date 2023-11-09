import {AppContext as ArgoAppContext, NavigationApi, NotificationsApi, PopupApi} from 'argo-ui';
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
