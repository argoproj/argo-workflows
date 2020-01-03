import * as React from 'react';

import {LogsViewerProps} from 'argo-ui/src/components/logs-viewer/logs-viewer';
import {LogsViewer} from 'argo-ui/src/index';

require('argo-ui/src/components/logs-viewer/logs-viewer.scss');

interface WorkflowLogsViewerProps extends LogsViewerProps {
    nodeId: string;
    container: string;
}

export class WorkflowLogsViewer extends React.Component<WorkflowLogsViewerProps> {
    public render() {
        return (
            <div>
                <h3>Logs</h3>
                <p>
                    {this.props.nodeId}/{this.props.container}
                </p>
                <LogsViewer {...this.props} />
            </div>
        );
    }
}
