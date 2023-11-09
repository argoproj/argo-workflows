import * as React from 'react';
import {Route, RouteComponentProps, Switch} from 'react-router';
import {EventSourceDetails} from './event-source-details/event-source-details';
import {EventSourceList} from './event-source-list/event-source-list';

export const EventSourceContainer = (props: RouteComponentProps<any>) => (
    <Switch>
        <Route exact={true} path={`${props.match.path}/:namespace?`} component={EventSourceList} />
        <Route exact={true} path={`${props.match.path}/:namespace/:name`} component={EventSourceDetails} />
    </Switch>
);
