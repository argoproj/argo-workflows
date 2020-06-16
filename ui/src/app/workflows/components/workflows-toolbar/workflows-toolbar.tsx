import * as React from 'react';
import {Workflow} from '../../../../models';
import {Consumer} from '../../../shared/context';
import * as Actions from '../../../shared/workflow-actions';

require('./workflows-toolbar.scss');

interface WorkflowsToolbarProps {
    selectedWorkflows: {[index: string]: Workflow};
    loadWorkflows: () => void;
    isDisabled: Actions.ActionDisabled;
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

    private performActionOnSelectedWorkflows(ctx: any, title: string, action: (name: string, namespace: string) => Promise<any>): void {
        this.confirmAction(title);
        for (const wfUID of Object.keys(this.props.selectedWorkflows)) {
            const wf = this.props.selectedWorkflows[wfUID];
            action(wf.metadata.name, wf.metadata.namespace)
                .catch(this.getHandleErrorFunction(title))
                .then(() => {
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
        const actions: any = Actions.WorkflowActions;
        const disabled: any = this.props.isDisabled;
        return Object.keys(actions).map(actionName => {
            const action = actions[actionName];
            return {
                title: action.title.charAt(0).toUpperCase() + action.title.slice(1),
                iconClassName: action.iconClassName,
                disabled: disabled[actionName],
                action: () => this.performActionOnSelectedWorkflows(ctx, action.title, action.action),
                className: action.title
            };
        });
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
