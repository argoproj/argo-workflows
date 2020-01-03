import * as React from 'react';

import * as models from '../../../../models';
import {services} from '../../../shared/services';

require('./workflow-logs-viewer.scss');

interface WorkflowLogsViewerProps {
    workflow: models.Workflow;
    nodeId: string;
    container: string;
    message?: React.ReactElement;
}

interface WorkflowLogsViewerState {
    error?: Error;
    lines: string[];
}

export class WorkflowLogsViewer extends React.Component<WorkflowLogsViewerProps, WorkflowLogsViewerState> {
    private logCoda: HTMLElement;

    constructor(props: WorkflowLogsViewerProps) {
        super(props);
        this.state = {lines: []};
    }

    public componentDidMount(): void {
        services.workflows.getContainerLogs(this.props.workflow, this.props.nodeId, this.props.container).subscribe(
            log => {
                if (log) {
                    this.setState(state => {
                        log.split('\n').forEach(line => {
                            state.lines.push(line);
                        });
                        return state;
                    });
                }
            },
            error => {
                this.setState({error});
            }
        );
    }

    public componentDidUpdate() {
        if (this.logCoda) {
            this.logCoda.scrollIntoView({behavior: 'auto'});
        }
    }

    public render() {
        return (
            <div className='workflow-logs-viewer'>
                <h3>Logs</h3>
                {this.props.message}
                <p>
                    <i className='fa fa-box' /> {this.props.nodeId}/{this.props.container}
                    {this.state.lines.length > 0 && <small className='muted'>{this.state.lines.length} line(s)</small>}
                </p>
                <div className='white-box'>
                    {this.state.error && (
                        <p>
                            <i className='fa fa-exclamation-triangle status-icon--failed' /> Failed to load logs: {this.state.error.message}
                        </p>
                    )}
                    {!this.state.error && this.state.lines.length === 0 && (
                        <p>
                            <i className='fa fa-circle-notch fa-spin' /> Loading...
                        </p>
                    )}
                    {this.state.lines.length > 0 && (
                        <div className='log-box'>
                            {this.state.lines.join('\n\r')}
                            <span
                                ref={el => {
                                    this.logCoda = el;
                                }}
                            />
                        </div>
                    )}
                </div>
            </div>
        );
    }
}
