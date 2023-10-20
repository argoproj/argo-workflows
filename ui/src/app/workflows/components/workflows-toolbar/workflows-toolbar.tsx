import {NotificationType} from 'argo-ui';
import * as React from 'react';
import {isArchivedWorkflow, isWorkflowInCluster, Workflow} from '../../../../models';
import {Consumer, ContextApis} from '../../../shared/context';
import {services} from '../../../shared/services';
import * as Actions from '../../../shared/workflow-operations-map';
import {WorkflowOperation, WorkflowOperationAction, WorkflowOperationName} from '../../../shared/workflow-operations-map';

require('./workflows-toolbar.scss');

interface WorkflowsToolbarProps {
    selectedWorkflows: Map<string, Workflow>;
    loadWorkflows: () => void;
    isDisabled: Actions.OperationDisabled;
    clearSelection: () => void;
}

interface WorkflowGroupAction extends WorkflowOperation {
    groupIsDisabled: boolean;
    className: string;
    groupAction: () => Promise<any>;
}

export class WorkflowsToolbar extends React.Component<WorkflowsToolbarProps, {}> {
    constructor(props: WorkflowsToolbarProps) {
        super(props);
    }

    public render() {
        return (
            <Consumer>
                {ctx => (
                    <div className={`workflows-toolbar ${this.getNumberSelected() === 0 ? 'hidden' : ''}`}>
                        <div className='workflows-toolbar__count'>
                            {this.getNumberSelected() === 0 ? 'No' : this.getNumberSelected()}
                            &nbsp;workflow{this.getNumberSelected() === 1 ? '' : 's'} selected
                        </div>
                        <div className='workflows-toolbar__actions'>{this.renderActions(ctx)}</div>
                    </div>
                )}
            </Consumer>
        );
    }

    private getNumberSelected(): number {
        return this.props.selectedWorkflows.size;
    }

    private async performActionOnSelectedWorkflows(ctx: ContextApis, title: string, action: WorkflowOperationAction): Promise<any> {
        const confirmed = await ctx.popup.confirm('Confirm', `Are you sure you want to ${title.toLowerCase()} all selected workflows?`);
        if (!confirmed) {
            return Promise.resolve(false);
        }

        let deleteArchived = false;
        if (title === 'DELETE') {
            for (const entry of this.props.selectedWorkflows) {
                if (isArchivedWorkflow(entry[1])) {
                    deleteArchived = await ctx.popup.confirm('Confirm', 'Do you also want to delete them from the Archived Workflows database?');
                    break;
                }
            }
        }

        const promises: Promise<any>[] = [];
        this.props.selectedWorkflows.forEach((wf: Workflow) => {
            if (title === 'DELETE') {
                // The ones without archivalStatus label or with 'Archived' labels are the live workflows.
                if (isWorkflowInCluster(wf)) {
                    promises.push(
                        services.workflows.delete(wf.metadata.name, wf.metadata.namespace).catch(reason =>
                            ctx.notifications.show({
                                content: `Unable to delete workflow ${wf.metadata.name} in the cluster: ${reason.toString()}`,
                                type: NotificationType.Error
                            })
                        )
                    );
                }
                if (deleteArchived && isArchivedWorkflow(wf)) {
                    promises.push(
                        services.workflows.deleteArchived(wf.metadata.uid, wf.metadata.namespace).catch(reason =>
                            ctx.notifications.show({
                                content: `Unable to delete workflow ${wf.metadata.name} in database: ${reason.toString()}`,
                                type: NotificationType.Error
                            })
                        )
                    );
                }
            } else {
                promises.push(
                    action(wf).catch(reason => {
                        this.props.loadWorkflows();
                        ctx.notifications.show({
                            content: `Unable to ${title} workflow: ${reason.content.toString()}`,
                            type: NotificationType.Error
                        });
                    })
                );
            }
        });
        return Promise.all(promises);
    }

    private renderActions(ctx: ContextApis): JSX.Element[] {
        const actionButtons = [];
        const actions: any = Actions.WorkflowOperationsMap;
        const disabled = this.props.isDisabled;
        const groupActions: WorkflowGroupAction[] = Object.keys(actions).map((actionName: WorkflowOperationName) => {
            const action = actions[actionName];
            return {
                title: action.title,
                iconClassName: action.iconClassName,
                groupIsDisabled: disabled[actionName],
                action,
                groupAction: async () => {
                    const confirmed = await this.performActionOnSelectedWorkflows(ctx, action.title, action.action);
                    if (!confirmed) {
                        return;
                    }

                    this.props.clearSelection();
                    ctx.notifications.show({
                        content: `Performed '${action.title}' on selected workflows.`,
                        type: NotificationType.Success
                    });
                    this.props.loadWorkflows();
                },
                className: action.title,
                disabled: () => false
            } as WorkflowGroupAction;
        });
        for (const groupAction of groupActions) {
            actionButtons.push(
                <button
                    key={groupAction.title}
                    onClick={() => {
                        groupAction.groupAction().catch();
                    }}
                    className={`workflows-toolbar__actions--${groupAction.className} workflows-toolbar__actions--action`}
                    disabled={this.getNumberSelected() === 0 || groupAction.groupIsDisabled}>
                    <i className={groupAction.iconClassName} />
                    &nbsp;{groupAction.title}
                </button>
            );
        }
        return actionButtons;
    }
}
