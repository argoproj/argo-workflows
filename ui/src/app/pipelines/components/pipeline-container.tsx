import * as React from 'react';
import {Route, RouteComponentProps, Switch} from 'react-router';
import {PipelineDetails} from './pipeline-details/pipeline-details';
import {PipelineList} from './pipeline-list/pipeline-list';

export const PipelineContainer = (props: RouteComponentProps<any>) => (
    <Switch>
        <Route exact={true} path={`${props.match.path}/:namespace?`} component={PipelineList} />
        <Route exact={true} path={`${props.match.path}/:namespace/:name`} component={PipelineDetails} />
    </Switch>
);
