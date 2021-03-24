import * as React from 'react';
import {Route, RouteComponentProps, Switch} from 'react-router';
import {Reports} from './reports';

export const ReportsContainer = (props: RouteComponentProps<any>) => (
    <Switch>
        <Route exact={true} path={`${props.match.path}/:namespace?`} component={Reports} />
    </Switch>
);
