import {Checkbox} from 'argo-ui/src/components/checkbox';
import * as React from 'react';
import {useState} from 'react';

interface Props {
    onKeepWorkflowsChange: (keepWorkflows: boolean) => void;
}

export function CronWorkflowDeleteConfirmation({onKeepWorkflowsChange}: Props) {
    const [keepWorkflows, setKeepWorkflows] = useState(false);

    const updateKeepWorkflows = (value: boolean) => {
        setKeepWorkflows(value);
        onKeepWorkflowsChange(value);
    };

    return (
        <>
            <p>Are you sure you want to delete this CronWorkflow?</p>
            <p>Deleting it also deletes all non-archived Workflows it created.</p>
            <div>
                <Checkbox id='keep-cron-workflow-workflows' checked={keepWorkflows} onChange={updateKeepWorkflows} />
                <label htmlFor='keep-cron-workflow-workflows'>Keep Workflows created by this CronWorkflow</label>
            </div>
        </>
    );
}
