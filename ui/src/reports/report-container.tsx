import * as React from 'react';
import {Route, Routes} from 'react-router-dom';

import {Loading} from '../shared/components/loading';

export function ReportsContainer() {
    return (
        <Routes>
            <Route path=':namespace?' element={<SuspenseReports />} />
        </Routes>
    );
}

// lazy load Reports as it is infrequently used and imports large Chart components (which can be split into a separate bundle)
const LazyReports = React.lazy(() => import(/* webpackChunkName: "reports" */ './reports'));

function SuspenseReports() {
    return (
        <React.Suspense fallback={<Loading />}>
            <LazyReports />
        </React.Suspense>
    );
}
