import * as React from 'react';
import {Route, RouteComponentProps, Switch} from 'react-router';
import {CronWorkflowDetails} from './cron-workflow-details/cron-workflow-details';
import {CronWorkflowList} from './cron-workflow-list/cron-workflow-list';

export const CronWorkflowContainer = (props: RouteComponentProps<any>) => (
    <Switch>
        <Route exact={true} path={`${props.match.path}/:namespace?`} component={CronWorkflowList} />
        <Route exact={true} path={`${props.match.path}/:namespace/:name`} component={CronWorkflowDetails} />
    </Switch>
);
