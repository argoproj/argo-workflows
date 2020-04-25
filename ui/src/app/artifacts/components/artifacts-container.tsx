import * as React from 'react';
import {Route, RouteComponentProps, Switch} from 'react-router';
import {ArtifactsList} from './artifacts-list/artifacts-list';

export const ArtifactsContainer = (props: RouteComponentProps<any>) => (
    <Switch>
        <Route exact={true} path={`${props.match.path}/:namespace?`} component={ArtifactsList} />
    </Switch>
);
