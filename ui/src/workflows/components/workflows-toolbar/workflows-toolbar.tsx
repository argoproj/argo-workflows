import {NotificationType} from 'argo-ui/src/components/notifications/notifications';
import * as React from 'react';
import {useContext, useMemo} from 'react';

import {Context} from '../../../shared/context';
import {getErrorMessage} from '../../../shared/errors';
import {isArchivedWorkflow, isWorkflowInCluster, Workflow} from '../../../shared/models';
import {services} from '../../../shared/services';
import * as Actions from '../../../shared/workflow-operations-map';
import {WorkflowOperation, WorkflowOperationAction, WorkflowOperationName} from '../../../shared/workflow-operations-map';

import './workflows-toolbar.scss';

interface WorkflowsToolbarProps {
    selectedWorkflows: Map<string, Workflow>;
    disabledActions: Actions.OperationDisabled;
    clearSelection: () => void;
    loadWorkflows: () => void;
}

interface WorkflowsOperation extends WorkflowOperation {
    isDisabled: boolean;
    action: () => Promise<any>;
}

export function WorkflowsToolbar(props: WorkflowsToolbarProps) {
    const {popup, notifications} = useContext(Context);
    const numberSelected = props.selectedWorkflows.size;

    const operations = useMemo<WorkflowsOperation[]>(() => {
        const actions: any = Actions.WorkflowOperationsMap;

        return Object.keys(actions).map((actionName: WorkflowOperationName) => {
            const action = actions[actionName];
            return {
                title: action.title,
                iconClassName: action.iconClassName,
                isDisabled: props.disabledActions[actionName],
                action: async () => {
                    const confirmed = await popup.confirm('Confirm', `Are you sure you want to ${action.title.toLowerCase()} all selected workflows?`);
                    if (!confirmed) {
                        return;
                    }

                    let deleteArchived = false;
                    if (action.title === 'DELETE') {
                        // check if there are archived workflows to delete
                        for (const entry of props.selectedWorkflows) {
                            if (isArchivedWorkflow(entry[1])) {
                                deleteArchived = await popup.confirm('Confirm', 'Do you also want to delete them from the Archived Workflows database?');
                                break;
                            }
                        }
                    }

                    const succeeded = await performActionOnSelectedWorkflows(action.title, action.action, deleteArchived);

                    props.clearSelection();
                    // each failure has already shown its own error, do not also report success
                    if (succeeded.every(ok => ok)) {
                        notifications.show({
                            content: `Performed '${action.title}' on selected workflows.`,
                            type: NotificationType.Success
                        });
                    }
                    props.loadWorkflows();
                },
                disabled: () => false
            } as WorkflowsOperation;
        });
    }, [props.selectedWorkflows]);

    // resolves to whether each action succeeded
    async function performActionOnSelectedWorkflows(title: string, action: WorkflowOperationAction, deleteArchived: boolean): Promise<boolean[]> {
        const promises: Promise<boolean>[] = [];
        const failed = (content: string) => {
            notifications.show({content, type: NotificationType.Error});
            return false;
        };
        props.selectedWorkflows.forEach((wf: Workflow) => {
            if (title === 'DELETE') {
                // The ones without archivalStatus label or with 'Archived' labels are the live workflows.
                if (isWorkflowInCluster(wf)) {
                    promises.push(
                        services.workflows.delete(wf.metadata.name, wf.metadata.namespace).then(
                            () => true,
                            reason => failed(`Unable to delete workflow ${wf.metadata.name} in the cluster: ${getErrorMessage(reason)}`)
                        )
                    );
                }
                if (deleteArchived && isArchivedWorkflow(wf)) {
                    promises.push(
                        services.workflows.deleteArchived(wf.metadata.uid, wf.metadata.namespace).then(
                            () => true,
                            reason => failed(`Unable to delete workflow ${wf.metadata.name} in database: ${getErrorMessage(reason)}`)
                        )
                    );
                }
            } else {
                promises.push(
                    action(wf).then(
                        () => true,
                        reason => failed(`Unable to ${title.toLowerCase()} workflow ${wf.metadata.name}: ${getErrorMessage(reason)}`)
                    )
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
                {operations.map(operation => {
                    return (
                        <button
                            key={operation.title}
                            onClick={operation.action}
                            className={`workflows-toolbar__actions--${operation.title} workflows-toolbar__actions--action`}
                            disabled={numberSelected === 0 || operation.isDisabled}>
                            <i className={operation.iconClassName} />
                            &nbsp;{operation.title}
                        </button>
                    );
                })}
            </div>
        </div>
    );
}
