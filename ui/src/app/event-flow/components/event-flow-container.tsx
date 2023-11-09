import * as React from 'react';
import {Route, RouteComponentProps, Switch} from 'react-router';
import {EventFlowPage} from './event-flow-details/event-flow-page';

export const EventFlowContainer = (props: RouteComponentProps<any>) => (
    <Switch>
        <Route path={`${props.match.path}/:namespace?`} component={EventFlowPage} />
    </Switch>
);
