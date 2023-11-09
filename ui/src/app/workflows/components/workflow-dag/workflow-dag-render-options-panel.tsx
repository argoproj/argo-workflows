import * as React from 'react';
import {WorkflowDagRenderOptions} from './workflow-dag';

export function WorkflowDagRenderOptionsPanel(props: WorkflowDagRenderOptions & {onChange: (changed: WorkflowDagRenderOptions) => void}) {
    function workflowDagRenderOptions() {
        return props as WorkflowDagRenderOptions;
    }

    return (
        <>
            <a
                onClick={() =>
                    props.onChange({
                        ...workflowDagRenderOptions(),
                        showArtifacts: !workflowDagRenderOptions().showArtifacts
                    })
                }
                className={workflowDagRenderOptions().showArtifacts ? 'active' : ''}
                title='Toggle artifacts'>
                <i className='fa fa-file-alt' />
            </a>
            <a
                onClick={() =>
                    props.onChange({
                        ...workflowDagRenderOptions(),
                        expandNodes: new Set()
                    })
                }
                title='Collapse all nodes'>
                <i className='fa fa-compress fa-fw' data-fa-transform='rotate-45' />
            </a>
            <a
                onClick={() =>
                    props.onChange({
                        ...workflowDagRenderOptions(),
                        expandNodes: new Set(['*'])
                    })
                }
                title='Expand all nodes'>
                <i className='fa fa-expand fa-fw' data-fa-transform='rotate-45' />
            </a>
        </>
    );
}
