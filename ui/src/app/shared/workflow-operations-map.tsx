import {NodePhase, Workflow} from '../../models';
import {services} from './services';
import {WorkflowDeleteResponse} from './services/responses';
import {Utils} from './utils';
import {useState} from "react";
import * as React from "react";
import {uiUrl} from "./base";

export type OperationDisabled = {
    [action in WorkflowOperationName]: boolean;
};

type WorkflowOperationName = 'RETRY' | 'RESUBMIT' | 'SUSPEND' | 'RESUME' | 'STOP' | 'TERMINATE' | 'DELETE';

export interface WorkflowOperation {
    title: WorkflowOperationName;
    action: WorkflowOperationAction;
    iconClassName: string;
    disabled: (wf: Workflow) => boolean;
}

export type WorkflowOperationAction = (wf: Workflow) => Promise<Workflow | WorkflowDeleteResponse>;

export interface WorkflowOperations {
    [name: string]: WorkflowOperation;
}

let globalDeleteArchived = false;

const DeleteCheck = (props: {isWfInDB: boolean; isWfInCluster: boolean}) => {
    // The local states are created intentionally so that the checkbox works as expected
    const [da, sda] = useState(false);
    if (props.isWfInDB && props.isWfInCluster) {
        return (
            <>
                <p>Are you sure you want to delete this workflow?</p>
                <div className='workflows-list__status'>
                    <input
                        type='checkbox'
                        className='workflows-list__status--checkbox'
                        checked={da}
                        onClick={() => {
                            sda(!da);
                            globalDeleteArchived = !globalDeleteArchived;
                        }}
                        id='delete-check'
                    />
                    <label htmlFor='delete-check'>Delete in database</label>
                </div>
            </>
        );
    } else {
        return (
            <>
                <p>Are you sure you want to delete this workflow?</p>
            </>
        );
    }
};

export const WorkflowOperationsMap: WorkflowOperations = {
    RETRY: {
        title: 'RETRY',
        iconClassName: 'fa fa-undo',
        disabled: (wf: Workflow) => {
            const workflowPhase: NodePhase = wf && wf.status ? wf.status.phase : undefined;
            return workflowPhase === undefined || !(workflowPhase === 'Failed' || workflowPhase === 'Error');
        },
        action: (wf: Workflow) => services.workflows.retry(wf.metadata.name, wf.metadata.namespace)
    },
    RESUBMIT: {
        title: 'RESUBMIT',
        iconClassName: 'fa fa-plus-circle',
        disabled: () => false,
        action: (wf: Workflow) => services.workflows.resubmit(wf.metadata.name, wf.metadata.namespace)
    },
    SUSPEND: {
        title: 'SUSPEND',
        iconClassName: 'fa fa-pause',
        disabled: (wf: Workflow) => !Utils.isWorkflowRunning(wf) || Utils.isWorkflowSuspended(wf),
        action: (wf: Workflow) => services.workflows.suspend(wf.metadata.name, wf.metadata.namespace)
    },
    RESUME: {
        title: 'RESUME',
        iconClassName: 'fa fa-play',
        disabled: (wf: Workflow) => !Utils.isWorkflowSuspended(wf),
        action: (wf: Workflow) => services.workflows.resume(wf.metadata.name, wf.metadata.namespace, null)
    },
    STOP: {
        title: 'STOP',
        iconClassName: 'fa fa-stop-circle',
        disabled: (wf: Workflow) => !Utils.isWorkflowRunning(wf),
        action: (wf: Workflow) => services.workflows.stop(wf.metadata.name, wf.metadata.namespace)
    },
    TERMINATE: {
        title: 'TERMINATE',
        iconClassName: 'fa fa-times-circle',
        disabled: (wf: Workflow) => !Utils.isWorkflowRunning(wf),
        action: (wf: Workflow) => services.workflows.terminate(wf.metadata.name, wf.metadata.namespace)
    },
    DELETE: {
        title: 'DELETE',
        iconClassName: 'fa fa-trash',
        disabled: () => false,
        action: (wf: Workflow) => {
            popup
                .confirm('Confirm', () => <DeleteCheck isWfInDB={isWfInDB} isWfInCluster={isWfInCluster} />)
                .then(yes => {
                    if (yes) {
                        if (isWfInCluster) {
                            services.workflows
                                .delete(workflow.metadata.name, workflow.metadata.namespace)
                                .then(() => {
                                    setIsWfInCluster(false);
                                })
                                .catch(setError);
                        }
                        if (isWfInDB && (globalDeleteArchived || !isWfInCluster)) {
                            services.workflows
                                .deleteArchived(workflow.metadata.uid, workflow.metadata.namespace)
                                .then(() => {
                                    setIsWfInDB(false);
                                })
                                .catch(setError);
                        }
                    }
                });
        }
    }
};
