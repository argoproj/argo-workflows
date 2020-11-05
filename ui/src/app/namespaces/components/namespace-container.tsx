import * as React from 'react';
import {Route, RouteComponentProps, Switch} from 'react-router';
import {NamespaceDetails} from './namespace-details/namespace-details';

export const NamespaceContainer = (props: RouteComponentProps<any>) => (
    <Switch>
        <Route exact={true} path={`${props.match.path}/:namespace`} component={NamespaceDetails} />
    </Switch>
);
