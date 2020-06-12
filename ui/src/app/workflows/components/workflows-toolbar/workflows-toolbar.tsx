import * as React from 'react';
import {Workflow} from '../../../../models';
import {Consumer, ContextApis} from '../../../shared/context';
import {services} from '../../../shared/services';

require('./workflows-toolbar.scss');

interface WorkflowsToolbarProps {
    selectedWorkflows: {[index: string]: Workflow};
}

export class WorkflowsToolbar extends React.Component<WorkflowsToolbarProps, {}> {
    constructor(props: WorkflowsToolbarProps) {
        super(props);
        this.deleteSelectedWorkflows = this.deleteSelectedWorkflows.bind(this);
    }

    public render() {
        return (
            <Consumer>
                {ctx => (
                    <div className='workflows-toolbar'>
                        <div className='workflows-toolbar__count'>{this.getNumberSelected()} workflows selected</div>
                        <div className='workflows-toolbar__actions'>
                            <button onClick={() => this.deleteSelectedWorkflows(ctx)} className='workflows-toolbar__actions--delete'>
                                Delete Selected&nbsp;
                                <i className='fas fa-trash-alt' />
                            </button>
                        </div>
                    </div>
                )}
            </Consumer>
        )
    }

    private getNumberSelected(): number {
        return Object.keys(this.props.selectedWorkflows).length;
    }

    private deleteSelectedWorkflows(ctx: ContextApis): void {
        if (!confirm('Are you sure you want to delete all selected workflows?')) {
            return;
        }
        for (const wfUID of Object.keys(this.props.selectedWorkflows)) {
            const wf = this.props.selectedWorkflows[wfUID];
            services.workflows
                .delete(wf.metadata.name, wf.metadata.namespace)
                .then(() => {
                    this.setState({ selectedWorkflows: {}}); 
                    ctx.navigation.goto('/');
                })
                .catch((err) => {
                    // TODO: Error handling
                });
        }
    }
}
