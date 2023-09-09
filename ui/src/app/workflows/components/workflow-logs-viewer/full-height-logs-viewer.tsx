import * as React from 'react';

import {LogsViewer} from 'argo-ui';
import {LogsViewerProps} from 'argo-ui/src/components/logs-viewer/logs-viewer';

require('./workflow-logs-viewer.scss');

export function FullHeightLogsViewer(props: LogsViewerProps) {
    const ref = React.useRef(null);
    const [height, setHeight] = React.useState<number>(null);
    const {source} = props;

    React.useEffect(() => {
        const parentElement = ref.current!.parentElement;
        setHeight(parentElement.getBoundingClientRect().height);
    }, [ref]);

    return (
        <div ref={ref} style={{height}} className='log-box'>
            {height && <LogsViewer source={source} />}
        </div>
    );
}
