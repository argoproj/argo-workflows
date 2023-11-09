import * as React from 'react';
import {Route, RouteComponentProps, Switch} from 'react-router';
import {SensorDetails} from './sensor-details/sensor-details';
import {SensorList} from './sensor-list/sensor-list';

export const SensorsContainer = (props: RouteComponentProps<any>) => (
    <Switch>
        <Route exact={true} path={`${props.match.path}/:namespace?`} component={SensorList} />
        <Route exact={true} path={`${props.match.path}/:namespace/:name`} component={SensorDetails} />
    </Switch>
);
