import * as React from 'react';
import {Workflow} from '../../../../models';
import {Consumer} from '../../../shared/context';
import * as Actions from '../../../shared/workflow-actions';

require('./workflows-toolbar.scss');

interface WorkflowsToolbarProps {
    selectedWorkflows: {[index: string]: Workflow};
    loadWorkflows: () => void;
    canSuspendSelected: boolean;
}

interface WorkflowsToolbarState {
    message: string;
}

interface WorkflowGroupAction {
    action: () => void;
    title: string;
    disabled: boolean;
    iconClassName: string;
    className: string;
}

export class WorkflowsToolbar extends React.Component<WorkflowsToolbarProps, WorkflowsToolbarState> {
    constructor(props: WorkflowsToolbarProps) {
        super(props);
        this.state = {
            message: ''
        };
    }

    public render() {
        return (
            <Consumer>
                {ctx => (
                    <div className='workflows-toolbar'>
                        <div className='workflows-toolbar__count'>
                            {this.noneSelected() ? 'No' : this.getNumberSelected()}
                            &nbsp;workflow{this.getNumberSelected() === 1 ? '' : 's'} selected
                        </div>
                        <div className='workflows-toolbar__message'>{this.state.message}</div>
                        <div className='workflows-toolbar__actions'>{this.renderActions(ctx)}</div>
                    </div>
                )}
            </Consumer>
        );
    }

    private noneSelected(): boolean {
        return Object.keys(this.props.selectedWorkflows).length < 1;
    }

    private getNumberSelected(): number {
        return Object.keys(this.props.selectedWorkflows).length;
    }

    private performActionOnSelectedWorkflows(ctx: any, title: string, action: (params: Actions.WorkflowActionParams) => Promise<any>): void {
        this.confirmAction(title);
        for (const wfUID of Object.keys(this.props.selectedWorkflows)) {
            const wf = this.props.selectedWorkflows[wfUID];
            action({
                ctx,
                name: wf.metadata.name,
                namespace: wf.metadata.namespace,
                handleError: this.getHandleErrorFunction(title)
            }).then(() => {
                this.setState({message: `Successfully performed '${title}' on selected workflows.`});
                this.props.loadWorkflows();
            });
        }
    }

    private confirmAction(title: string): void {
        if (!confirm(`Are you sure you want to ${title} all selected workflows?`)) {
            return;
        }
        return;
    }

    private getHandleErrorFunction(title: string): () => void {
        return () => {
            this.setState({message: `Could not ${title} selected workflows`});
            this.props.loadWorkflows();
        };
    }

    private getActions(ctx: any): WorkflowGroupAction[] {
        return [
            {
                action: () => this.performActionOnSelectedWorkflows(ctx, 'retry', Actions.retryWorkflow),
                disabled: this.props.canSuspendSelected,
                iconClassName: 'fas fa-redo-alt',
                title: 'Retry',
                className: 'retry'
            },
            {
                action: () => this.performActionOnSelectedWorkflows(ctx, 'resubmit', Actions.resubmitWorkflow),
                disabled: false,
                iconClassName: 'fas fa-plus-circle',
                title: 'Resubmit',
                className: 'resubmit'
            },
            {
                action: () => this.performActionOnSelectedWorkflows(ctx, 'suspend', Actions.suspendWorkflow),
                disabled: !this.props.canSuspendSelected,
                iconClassName: 'fas fa-pause',
                title: 'Suspend',
                className: 'suspend'
            },
            {
                action: () => this.performActionOnSelectedWorkflows(ctx, 'resume', Actions.resumeWorkflow),
                disabled: !this.props.canSuspendSelected,
                iconClassName: 'fas fa-play',
                title: 'Resume',
                className: 'resume'
            },
            {
                action: () => this.performActionOnSelectedWorkflows(ctx, 'delete', Actions.deleteWorkflow),
                disabled: false,
                iconClassName: 'fas fa-trash-alt',
                title: 'Delete',
                className: 'delete'
            }
        ];
    }

    private renderActions(ctx: any): JSX.Element[] {
        const actionButtons = [];
        for (const action of this.getActions(ctx)) {
            actionButtons.push(
                <button
                    key={action.title}
                    onClick={action.action}
                    className={`workflows-toolbar__actions--${action.className} workflows-toolbar__actions--action`}
                    disabled={this.noneSelected() || action.disabled}>
                    <i className={action.iconClassName} />
                    &nbsp;{action.title} Selected
                </button>
            );
        }
        return actionButtons;
    }
}
