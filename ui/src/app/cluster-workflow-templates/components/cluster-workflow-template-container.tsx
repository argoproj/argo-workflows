import * as React from 'react';
import {Route, RouteComponentProps, Switch} from 'react-router';
import {ClusterWorkflowTemplateDetails} from './cluster-workflow-template-details/cluster-workflow-template-details';
import {ClusterWorkflowTemplateList} from './cluster-workflow-template-list/cluster-workflow-template-list';

export const ClusterWorkflowTemplateContainer = (props: RouteComponentProps<any>) => (
    <Switch>
        <Route exact={true} path={`${props.match.path}`} component={ClusterWorkflowTemplateList} />
        <Route exact={true} path={`${props.match.path}/:name`} component={ClusterWorkflowTemplateDetails} />
    </Switch>
);
