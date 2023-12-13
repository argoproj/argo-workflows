import * as React from 'react';
import {useEffect, useRef, useState} from 'react';

import {LogsViewer} from 'argo-ui';
import {LogsViewerProps} from 'argo-ui/src/components/logs-viewer/logs-viewer';

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
            {height && <LogsViewer source={source} />}
        </div>
    );
}
