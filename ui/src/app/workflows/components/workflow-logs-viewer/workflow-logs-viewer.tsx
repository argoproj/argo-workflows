import * as React from 'react';

import {Observable, Subscription} from 'rxjs';
import * as models from '../../../../models';
import {ErrorNotice} from '../../../shared/components/error-notice';
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
    lineCount: number;
}

function identity<T>(value: T) {
    return () => value;
}

export class WorkflowLogsViewer extends React.Component<WorkflowLogsViewerProps, WorkflowLogsViewerState> {
    private subscription: Subscription | null = null;
    private logsObservable?: Observable<string>;

    constructor(props: WorkflowLogsViewerProps) {
        super(props);

        this.state = {lineCount: 0, loaded: false};
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
                    {this.state.lineCount > 0 && <small className='muted'> {this.state.lineCount} line(s)</small>}
                </p>

                {this.state.error && <ErrorNotice error={this.state.error} onReload={() => this.refreshStream()} />}
                <div className='white-box'>
                    {this.isWaitingForData() && (
                        <p>
                            <i className='fa fa-circle-notch fa-spin' /> Waiting for data...
                        </p>
                    )}
                    {!this.state.error && this.podHasNoLogs() && <p>Pod did not output any logs.</p>}
                    {this.state.lineCount > 0 && this.logsObservable && (
                        <div className='log-box'>
                            <FullHeightLogsViewer
                                source={{
                                    key: `${this.props.workflow.metadata.name}-${this.props.container}`,
                                    loadLogs: identity(this.logsObservable),
                                    shouldRepeat: () => false
                                }}
                            />
                        </div>
                    )}
                </div>
                {this.state.lineCount === 0 && (
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

        this.setState({lineCount: 0, loaded: false, error: undefined});

        const source = services.workflows.getContainerLogs(this.props.workflow, this.props.nodeId, this.props.container, this.props.archived).map(line => line + '\n');

        this.logsObservable = source.publishReplay().refCount();
        this.subscription = this.logsObservable.subscribe(
            log => {
                this.setState(state => {
                    return {...state, lineCount: state.lineCount + 1};
                });
            },
            error => {
                this.setState({error, loaded: true});
            }
        );
    }

    private podHasNoLogs() {
        return !this.isWaitingForData() && this.state.lineCount === 0;
    }

    private isWaitingForData() {
        return this.state.lineCount === 0 && (this.isCurrentNodeRunningOrPending() || !this.state.loaded);
    }

    private isCurrentNodeRunningOrPending(): boolean {
        return this.props.workflow.status.nodes[this.props.nodeId].phase === 'Running' || this.props.workflow.status.nodes[this.props.nodeId].phase === 'Pending';
    }
}
