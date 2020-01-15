import * as React from 'react';
import {Route, RouteComponentProps, Switch} from 'react-router';
import {ArchivedWorkflowDetails} from './archived-workflow-details/archived-workflow-details';
import {ArchivedWorkflowList} from './archived-workflow-list/archived-workflow-list';

export const ArchivedWorkflowContainer = (props: RouteComponentProps<any>) => (
    <Switch>
        <Route exact={true} path={`${props.match.path}`} component={ArchivedWorkflowList} />
        <Route exact={true} path={`${props.match.path}/:uid`} component={ArchivedWorkflowDetails} />
    </Switch>
);
