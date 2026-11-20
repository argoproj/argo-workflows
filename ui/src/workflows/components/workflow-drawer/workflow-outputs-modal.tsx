import * as React from 'react';

import {InlineTable} from '../../../shared/components/inline-table/inline-table';
import {Modal} from '../../../shared/components/modal/modal';
import {Outputs, Workflow} from '../../../shared/models';

export function hasOutputs(outputs?: Outputs): boolean {
    return !!outputs && (!!outputs.parameters?.length || !!outputs.artifacts?.length || outputs.result !== undefined || outputs.exitCode !== undefined);
}

// The root node's ID is the workflow name, and holds the entrypoint template's outputs.
export function entrypointOutputs(wf: Workflow): Outputs | undefined {
    return wf.status?.nodes?.[wf.metadata.name]?.outputs;
}

export function workflowHasOutputs(wf: Workflow): boolean {
    return hasOutputs(wf.status?.outputs) || hasOutputs(entrypointOutputs(wf));
}

function OutputsSection({title, outputs}: {title: string; outputs?: Outputs}) {
    if (!hasOutputs(outputs)) {
        return null;
    }
    const rows = [
        ...(outputs.parameters || []).map(p => ({left: <span>{p.name}</span>, right: <pre className='workflow-outputs-modal__value'>{p.value ?? ''}</pre>})),
        ...(outputs.artifacts || []).map(a => ({left: <span>{a.name}</span>, right: <span className='workflow-outputs-modal__artifact'>artifact</span>})),
        ...(outputs.result !== undefined ? [{left: <span>result</span>, right: <pre className='workflow-outputs-modal__value'>{outputs.result}</pre>}] : []),
        ...(outputs.exitCode !== undefined ? [{left: <span>exitCode</span>, right: <span>{outputs.exitCode}</span>}] : [])
    ];
    return (
        <div className='workflow-outputs-modal__section'>
            <h5>{title}</h5>
            <InlineTable rows={rows} />
        </div>
    );
}

export function WorkflowOutputsModal({workflow, dismiss}: {workflow: Workflow; dismiss: () => void}) {
    return (
        <Modal dismiss={dismiss}>
            <h4>Outputs of {workflow.metadata.name}</h4>
            <OutputsSection title='Global Outputs' outputs={workflow.status?.outputs} />
            <OutputsSection title='Entrypoint Outputs' outputs={entrypointOutputs(workflow)} />
        </Modal>
    );
}
