import * as React from 'react';
import {services} from '../../../shared/services';
import {Workflow} from '../../../../models';

require('./workflows-toolbar.scss');

interface WorkflowsToolbarProps {
    selectedWorkflows: {[index: string]: Workflow};
}

export class WorkflowsToolbar extends React.Component<WorkflowsToolbarProps, {}> {
    constructor(props: WorkflowsToolbarProps) {
        super(props);
    }

    public render() {
        return (
            <div className='workflows-toolbar'>
                <div className='workflows-toolbar__count'>
                    {this.getNumberSelected()} workflows selected
                </div>
                <div className='workflows-toolbar__actions'>
                    <div onClick={this.deleteSelectedWorkflows}>Delete Selected</div>
                </div>
            </div>
        )
    }

    private getNumberSelected(): number {
        return Object.keys(this.props.selectedWorkflows).length;
    }

    private deleteSelectedWorkflows(): void {
        for (const wfUID of Object.keys(this.props.selectedWorkflows)) {
            const wf = this.props.selectedWorkflows[wfUID];
            services.workflows
            .delete(wf.metadata.name, wf.metadata.namespace)
            .catch(() => {
                // TODO: Error handling
            });
        }
    }
}
