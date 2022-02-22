import {NotificationType} from 'argo-ui';
import * as PropTypes from 'prop-types';
import * as React from 'react';
import {Workflow} from '../../../../models';
import {AppContext, Consumer} from '../../../shared/context';
import * as Actions from '../../../shared/workflow-operations-map';
import {WorkflowOperation, WorkflowOperationAction} from '../../../shared/workflow-operations-map';

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
    public static contextTypes = {
        router: PropTypes.object,
        apis: PropTypes.object
    };

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

    private performActionOnSelectedWorkflows(ctx: any, title: string, action: WorkflowOperationAction): Promise<any> {
        if (!confirm(`Are you sure you want to ${title.toLowerCase()} all selected workflows?`)) {
            return Promise.resolve(false);
        }
        const promises: Promise<any>[] = [];
        this.props.selectedWorkflows.forEach((wf: Workflow) => {
            promises.push(
                action(wf).catch(() => {
                    this.props.loadWorkflows();
                    this.appContext.apis.notifications.show({
                        content: `Unable to ${title} workflow`,
                        type: NotificationType.Error
                    });
                })
            );
        });
        return Promise.all(promises);
    }

    private renderActions(ctx: any): JSX.Element[] {
        const actionButtons = [];
        const actions: any = Actions.WorkflowOperationsMap;
        const disabled: any = this.props.isDisabled;
        const groupActions: WorkflowGroupAction[] = Object.keys(actions).map(actionName => {
            const action = actions[actionName];
            return {
                title: action.title,
                iconClassName: action.iconClassName,
                groupIsDisabled: disabled[actionName],
                action,
                groupAction: () => {
                    return this.performActionOnSelectedWorkflows(ctx, action.title, action.action).then(confirmed => {
                        if (confirmed) {
                            this.props.clearSelection();
                            this.appContext.apis.notifications.show({
                                content: `Performed '${action.title}' on selected workflows.`,
                                type: NotificationType.Success
                            });
                            this.props.loadWorkflows();
                        }
                    });
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

    private get appContext(): AppContext {
        return this.context as AppContext;
    }
}
