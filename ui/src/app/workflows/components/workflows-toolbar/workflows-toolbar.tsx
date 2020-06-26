import {NotificationType} from 'argo-ui';
import * as PropTypes from 'prop-types';
import * as React from 'react';
import {Workflow} from '../../../../models';
import {AppContext, Consumer} from '../../../shared/context';
import * as Actions from '../../../shared/workflow-operations';
import {WorkflowOperation} from '../../../shared/workflow-operations';

require('./workflows-toolbar.scss');

interface WorkflowsToolbarProps {
    selectedWorkflows: {[index: string]: Workflow};
    loadWorkflows: () => void;
    isDisabled: Actions.OperationDisabled;
}

interface WorkflowGroupAction extends WorkflowOperation {
    groupIsDisabled: boolean;
    className: string;
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
        return Object.keys(this.props.selectedWorkflows).length;
    }

    private performActionOnSelectedWorkflows(ctx: any, title: string, action: (name: string, namespace: string) => Promise<any>): Promise<any> {
        this.confirmAction(title);
        const promises = [];
        for (const wfUID of Object.keys(this.props.selectedWorkflows)) {
            const wf = this.props.selectedWorkflows[wfUID];
            promises.push(
                action(wf.metadata.name, wf.metadata.namespace).catch(() => {
                    this.props.loadWorkflows();
                    this.appContext.apis.notifications.show({
                        content: `Unable to ${title} workflow`,
                        type: NotificationType.Error
                    });
                })
            );
        }
        return Promise.all(promises);
    }

    private confirmAction(title: string): void {
        if (!confirm(`Are you sure you want to ${title.toLowerCase()} all selected workflows?`)) {
            return;
        }
        return;
    }

    private renderActions(ctx: any): JSX.Element[] {
        const actionButtons = [];
        const actions: any = Actions.WorkflowOperations;
        const disabled: any = this.props.isDisabled;
        const groupActions: WorkflowGroupAction[] = Object.keys(actions).map(actionName => {
            const action = actions[actionName];
            return {
                title: action.title,
                iconClassName: action.iconClassName,
                groupIsDisabled: disabled[actionName],
                action: () => {
                    return this.performActionOnSelectedWorkflows(ctx, action.title, action.action).then(() => {
                        this.appContext.apis.notifications.show({
                            content: `Performed '${action.title}' on selected workflows.`,
                            type: NotificationType.Success
                        });
                        this.props.loadWorkflows();
                    });
                },
                className: action.title,
                disabled: () => false
            };
        });
        for (const action of groupActions) {
            actionButtons.push(
                <button
                    key={action.title}
                    onClick={action.action}
                    className={`workflows-toolbar__actions--${action.className} workflows-toolbar__actions--action`}
                    disabled={this.getNumberSelected() === 0 || action.groupIsDisabled}>
                    <i className={action.iconClassName} />
                    &nbsp;{action.title}
                </button>
            );
        }
        return actionButtons;
    }

    private get appContext(): AppContext {
        return this.context as AppContext;
    }
}
