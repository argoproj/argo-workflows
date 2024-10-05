import * as React from 'react';
import {useEffect, useRef, useState} from 'react';

import type {LogsViewerProps} from 'argo-ui/src/components/logs-viewer/logs-viewer';

import {Loading} from '../../../shared/components/loading';

import './workflow-logs-viewer.scss';

export function FullHeightLogsViewer(props: LogsViewerProps) {
    const ref = useRef(null);
    const [height, setHeight] = useState<number>(null);
    const {source} = props;

    useEffect(() => {
        const parentElement = ref.current!.parentElement;
        setHeight(parentElement.getBoundingClientRect().height);
    }, [ref]);

    return (
        <div ref={ref} style={{height}} className='log-box'>
            {height && <SuspenseLogsViewer source={source} />}
        </div>
    );
}

// lazy load LogsViewer as it imports a large component: xterm (which can be split into a separate bundle)
const LazyLogsViewer = React.lazy(async () => {
    // prefetch b/c logs are commonly used
    const module = await import(/* webpackPrefetch: true, webpackChunkName: "argo-ui-logs-viewer" */ 'argo-ui/src/components/logs-viewer/logs-viewer');
    return {default: module.LogsViewer}; // React.lazy requires a default import, so we create an intermediate module
});

function SuspenseLogsViewer(props: LogsViewerProps) {
    return (
        <React.Suspense fallback={<Loading />}>
            <LazyLogsViewer {...props} />
        </React.Suspense>
    );
}
