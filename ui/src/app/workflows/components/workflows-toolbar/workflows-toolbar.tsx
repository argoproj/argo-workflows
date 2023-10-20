import {NotificationType} from 'argo-ui';
import * as React from 'react';
import {useContext, useMemo} from 'react';
import {isArchivedWorkflow, isWorkflowInCluster, Workflow} from '../../../../models';
import {Context} from '../../../shared/context';
import {services} from '../../../shared/services';
import * as Actions from '../../../shared/workflow-operations-map';
import {WorkflowOperation, WorkflowOperationAction, WorkflowOperationName} from '../../../shared/workflow-operations-map';

require('./workflows-toolbar.scss');

interface WorkflowsToolbarProps {
    selectedWorkflows: Map<string, Workflow>;
    actionsIsDisabled: Actions.OperationDisabled;
    clearSelection: () => void;
}

interface WorkflowsOperation extends WorkflowOperation {
    isDisabled: boolean;
    workflowsAction: () => Promise<any>;
}

export function WorkflowsToolbar(props: WorkflowsToolbarProps) {
    const {popup, notifications} = useContext(Context);
    const numberSelected: number = props.selectedWorkflows.size;

    const groupAction = useMemo<WorkflowsOperation[]>(() => {
        const actions: any = Actions.WorkflowOperationsMap;
        const disabled = props.actionsIsDisabled;

        return Object.keys(actions).map((actionName: WorkflowOperationName) => {
            const action = actions[actionName];
            return {
                title: action.title,
                iconClassName: action.iconClassName,
                isDisabled: disabled[actionName],
                action,
                workflowsAction: async () => {
                    //check for action
                    const confirmed = await popup.confirm('Confirm', `Are you sure you want to ${action.title.toLowerCase()} all selected workflows?`);
                    if (!confirmed) {
                        return;
                    }

                    // check for delete from archived workflows
                    let deleteArchived = false;
                    if (action.title === 'DELETE') {
                        // check for delete workflows from archived workflows
                        for (const entry of props.selectedWorkflows) {
                            if (isArchivedWorkflow(entry[1])) {
                                deleteArchived = await popup.confirm('Confirm', 'Do you also want to delete them from the Archived Workflows database?');
                                break;
                            }
                        }
                    }

                    performActionOnSelectedWorkflows(action.title, action.action, deleteArchived);

                    props.clearSelection();
                    notifications.show({
                        content: `Performed '${action.title}' on selected workflows.`,
                        type: NotificationType.Success
                    });
                },
                disabled: () => false
            } as WorkflowsOperation;
        });
    }, [props.selectedWorkflows]);

    function performActionOnSelectedWorkflows(title: string, action: WorkflowOperationAction, deleteArchived: boolean): Promise<any> {
        const promises: Promise<any>[] = [];
        props.selectedWorkflows.forEach((wf: Workflow) => {
            if (title === 'DELETE') {
                // The ones without archivalStatus label or with 'Archived' labels are the live workflows.
                if (isWorkflowInCluster(wf)) {
                    promises.push(
                        services.workflows.delete(wf.metadata.name, wf.metadata.namespace).catch(reason =>
                            notifications.show({
                                content: `Unable to delete workflow ${wf.metadata.name} in the cluster: ${reason.toString()}`,
                                type: NotificationType.Error
                            })
                        )
                    );
                }
                if (deleteArchived && isArchivedWorkflow(wf)) {
                    promises.push(
                        services.workflows.deleteArchived(wf.metadata.uid, wf.metadata.namespace).catch(reason =>
                            notifications.show({
                                content: `Unable to delete workflow ${wf.metadata.name} in database: ${reason.toString()}`,
                                type: NotificationType.Error
                            })
                        )
                    );
                }
            } else {
                promises.push(
                    action(wf).catch(reason => {
                        notifications.show({
                            content: `Unable to ${title} workflow: ${reason.content.toString()}`,
                            type: NotificationType.Error
                        });
                    })
                );
            }
        });
        return Promise.all(promises);
    }

    return (
        <div className={`workflows-toolbar ${numberSelected === 0 ? 'hidden' : ''}`}>
            <div className='workflows-toolbar__count'>
                {numberSelected === 0 ? 'No' : numberSelected}
                &nbsp;workflow{numberSelected === 1 ? '' : 's'} selected
            </div>
            <div className='workflows-toolbar__actions'>
                {groupAction.map(action => {
                    return (
                        <button
                            key={action.title}
                            onClick={() => {
                                action.workflowsAction().catch();
                            }}
                            className={`workflows-toolbar__actions--${action.title} workflows-toolbar__actions--action`}
                            disabled={numberSelected === 0 || action.isDisabled}>
                            <i className={action.iconClassName} />
                            &nbsp;{action.title}
                        </button>
                    );
                })}
            </div>
        </div>
    );
}
