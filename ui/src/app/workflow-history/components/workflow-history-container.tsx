import * as React from 'react';
import {Route, RouteComponentProps, Switch} from 'react-router';
import {WorkflowHistoryList} from './workflow-history-list/workflow-history-list';

export const WorkflowHistoryContainer = (props: RouteComponentProps<any>) => (
    <Switch>
        <Route exact={true} path={`${props.match.path}`} component={WorkflowHistoryList} />
    </Switch>
);
