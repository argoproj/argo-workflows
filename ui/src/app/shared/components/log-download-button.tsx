import * as React from 'react';
import {Workflow} from '../../../models';
import {services} from '../services';

interface LogDownloadProps {
    workflows: Workflow[];
}

interface LogDownloadState {
    workflows: Workflow[];
}

export class LogDownloadButton extends React.Component<LogDownloadProps, LogDownloadState> {
    public static getDerivedStateFromProps(props: LogDownloadProps, currentState: LogDownloadState) {
        // This only updates when the length is different, because it's only used to show/hide the
        // button. The onSubmit handler for the form updates workflows before hitting the logs
        // endpoint.
        if (currentState.workflows.length !== props.workflows.length) {
            return {
                workflows: props.workflows
            };
        }

        return null;
    }

    constructor(props: Readonly<LogDownloadProps>) {
        super(props);
        this.state = {
            workflows: props.workflows
        };
    }

    public render() {
        if (this.state.workflows.length === 0) {
            return null;
        }

        // TODO: don't stringify until submit event, for performance.
        return (
            <form method='POST' action={services.workflows.getLogsMultipleUrl()} onSubmit={() => this.setState({workflows: this.props.workflows})}>
                <input type='hidden' name='workflows' value={JSON.stringify(this.state.workflows.map(wf => ({name: wf.metadata.name, namespace: wf.metadata.namespace})))} />
                <button type='submit' className='argo-button argo-button--base argo-button--sm'>
                    <i className='icon argo-icon-download' /> Download all logs
                </button>
            </form>
        );
    }
}
