import * as React from 'react';
import {Route, RouteComponentProps, Switch} from 'react-router';
import {WorkflowHistoryDetails} from './workflow-history-details/workflow-history-details';
import {WorkflowHistoryList} from './workflow-history-list/workflow-history-list';

export const WorkflowHistoryContainer = (props: RouteComponentProps<any>) => (
    <Switch>
        <Route exact={true} path={`${props.match.path}`} component={WorkflowHistoryList} />
        <Route exact={true} path={`${props.match.path}/:namespace/:uid`} component={WorkflowHistoryDetails} />
    </Switch>
);
