import * as React from 'react';

import {Subscription} from 'rxjs';
import * as models from '../../../../models';
import {services} from '../../../shared/services';
import {FullHeightLogsViewer} from './full-height-logs-viewer';

require('./workflow-logs-viewer.scss');

interface WorkflowLogsViewerProps {
    workflow: models.Workflow;
    nodeId: string;
    container: string;
    archived: boolean;
}

interface WorkflowLogsViewerState {
    error?: Error;
    loaded: boolean;
    lines: string[];
}

export class WorkflowLogsViewer extends React.Component<WorkflowLogsViewerProps, WorkflowLogsViewerState> {
    private subscription: Subscription = null;
    constructor(props: WorkflowLogsViewerProps) {
        super(props);
        this.state = {lines: [], loaded: false};
    }

    public componentDidMount(): void {
        this.refreshStream();
    }

    public componentWillUnmount(): void {
        this.ensureUnsubscribed();
    }

    public render() {
        return (
            <div className='workflow-logs-viewer'>
                <h3>Logs</h3>
                {this.props.archived && (
                    <p>
                        <i className='fa fa-exclamation-triangle' /> Logs for archived workflows may be overwritten by a more recent workflow with the same name.
                    </p>
                )}
                <p>
                    <i className='fa fa-box' /> {this.props.nodeId}/{this.props.container}
                    {this.state.lines.length > 0 && <small className='muted'> {this.state.lines.length} line(s)</small>}
                </p>
                <div className='white-box'>
                    {this.state.error && (
                        <p>
                            <i className='fa fa-exclamation-triangle status-icon--failed' /> Failed to load logs: {this.state.error.message}
                        </p>
                    )}
                    {!this.state.error && this.isWaitingForData() && (
                        <p>
                            <i className='fa fa-circle-notch fa-spin' /> Waiting for data...
                        </p>
                    )}
                    {!this.state.error && this.podHasNoLogs() && <p>Pod did not output any logs.</p>}
                    {this.state.lines.length > 0 && (
                        <div className='log-box'>
                            <FullHeightLogsViewer
                                source={{
                                    key: `${this.props.workflow.metadata.name}-${this.props.container}`,
                                    loadLogs: () => {
                                        return services.workflows.getContainerLogs(this.props.workflow, this.props.nodeId, this.props.container, this.props.archived).map(log => {
                                            return log ? log + '\n' : '';
                                        });
                                    },
                                    shouldRepeat: () => this.isCurrentNodeRunningOrPending()
                                }}
                            />
                        </div>
                    )}
                </div>
                {this.state.lines.length === 0 && (
                    <p>
                        Still waiting for data or an error? Try getting{' '}
                        <a href={services.workflows.getArtifactLogsUrl(this.props.workflow, this.props.nodeId, this.props.container, this.props.archived)}>
                            logs from the artifacts
                        </a>
                        .
                    </p>
                )}
            </div>
        );
    }

    private ensureUnsubscribed() {
        if (this.subscription) {
            this.subscription.unsubscribe();
        }
        this.subscription = null;
    }

    private refreshStream(): void {
        this.ensureUnsubscribed();
        this.subscription = services.workflows.getContainerLogs(this.props.workflow, this.props.nodeId, this.props.container, this.props.archived).subscribe(
            log => {
                this.setState(state => {
                    const newState = {...state, loaded: true};

                    if (log) {
                        newState.lines = newState.lines.concat(log.split('\n'));
                    }

                    return newState;
                });
            },
            error => {
                this.setState({error, loaded: true});
            }
        );
    }

    private podHasNoLogs() {
        return !this.isWaitingForData() && this.state.lines.length === 0;
    }

    private isWaitingForData() {
        return this.state.lines.length === 0 && (this.isCurrentNodeRunningOrPending() || !this.state.loaded);
    }

    private isCurrentNodeRunningOrPending(): boolean {
        return this.props.workflow.status.nodes[this.props.nodeId].phase === 'Running' || this.props.workflow.status.nodes[this.props.nodeId].phase === 'Pending';
    }
}
