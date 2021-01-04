import * as React from 'react';
import {Route, RouteComponentProps, Switch} from 'react-router';
import {WorkflowEventBindingsList} from './workflow-event-bindings-list/workflow-event-bindings-list';

export const WorkflowEventBindingsContainer = (props: RouteComponentProps<any>) => (
    <Switch>
        <Route exact={true} path={`${props.match.path}/:namespace?`} component={WorkflowEventBindingsList} />
    </Switch>
);
