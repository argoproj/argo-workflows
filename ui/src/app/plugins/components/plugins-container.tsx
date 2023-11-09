import * as React from 'react';
import {Route, RouteComponentProps, Switch} from 'react-router';
import {PluginList} from './plugin-list/plugin-list';

export const PluginsContainer = (props: RouteComponentProps<any>) => (
    <Switch>
        <Route exact={true} path={`${props.match.path}/:namespace?`} component={PluginList} />
    </Switch>
);
