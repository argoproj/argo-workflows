import * as React from 'react';
import {Route, RouteComponentProps, Switch} from 'react-router';
import {WorkflowEventBindings} from './workflow-event-bindings/workflow-event-bindings';

export const WorkflowEventBindingsContainer = (props: RouteComponentProps<any>) => (
    <Switch>
        <Route exact={true} path={`${props.match.path}/:namespace?`} component={WorkflowEventBindings} />
    </Switch>
);
