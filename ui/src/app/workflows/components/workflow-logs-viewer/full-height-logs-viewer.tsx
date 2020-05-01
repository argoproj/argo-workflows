import * as React from 'react';

import {LogsViewer} from 'argo-ui';
import {LogsViewerProps} from 'argo-ui/src/components/logs-viewer/logs-viewer';

export const FullHeightLogsViewer = (props: LogsViewerProps) => {
    const ref = React.useRef(null);
    const [height, setHeight] = React.useState<number>(null);
    const {source} = props;

    React.useEffect(() => {
        const parentElement = ref.current!.parentElement;
        setHeight(parentElement.getBoundingClientRect().height);
    }, [ref]);

    return (
        <div ref={ref} style={{height}}>
            {height && <LogsViewer source={source} />}
        </div>
    );
};
