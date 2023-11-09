import * as React from 'react';
import {Route, RouteComponentProps, Switch} from 'react-router';
import {ErrorNotice} from '../shared/components/error-notice';
import {WorkflowGraph} from './workflow-graph';
import {WorkflowStatusBadge} from './workflow-status-badge';

export const Widgets = (props: RouteComponentProps<any>) => (
    <Switch>
        <Route path={`${props.match.path}/workflow-graphs/:namespace`} component={WorkflowGraph} />
        <Route path={`${props.match.path}/workflow-status-badges/:namespace`} component={WorkflowStatusBadge} />
        <ErrorNotice error={new Error('Widget not found')} />
    </Switch>
);
