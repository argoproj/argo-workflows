import * as React from 'react';
import {Route, RouteComponentProps, Switch} from 'react-router';
import {EventsPage} from './events-details/events-page';

export const EventsContainer = (props: RouteComponentProps<any>) => (
    <Switch>
        <Route path={`${props.match.path}/:namespace?`} component={EventsPage} />
    </Switch>
);
