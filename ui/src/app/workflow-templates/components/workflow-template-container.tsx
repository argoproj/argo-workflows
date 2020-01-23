import * as React from 'react';
import {Route, RouteComponentProps, Switch} from 'react-router';
import {WorkflowTemplateDetails} from './workflow-template-details/workflow-template-details';
import {WorkflowTemplateList} from './workflow-template-list/workflow-template-list';

export const WorkflowTemplateContainer = (props: RouteComponentProps<any>) => (
    <Switch>
        <Route exact={true} path={`${props.match.path}/:namespace?`} component={WorkflowTemplateList} />
        <Route exact={true} path={`${props.match.path}/:namespace/:name`} component={WorkflowTemplateDetails} />
    </Switch>
);
