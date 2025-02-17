import * as React from 'react';
import {Route, RouteComponentProps, Switch} from 'react-router';
import {Loading} from '../../shared/components/loading';

export const ReportsContainer = (props: RouteComponentProps<any>) => (
    <Switch>
        <Route exact={true} path={`${props.match.path}/:namespace?`} component={SuspenseReports} />
    </Switch>
);

// lazy load Reports as it is infrequently used and imports large Chart components (which can be split into a separate bundle)
const LazyReports = React.lazy(() => import(/* webpackChunkName: "reports" */ './reports'));

function SuspenseReports(props: RouteComponentProps<any>) {
    return (
        <React.Suspense fallback={<Loading />}>
            <LazyReports {...props} />
        </React.Suspense>
    );
}
