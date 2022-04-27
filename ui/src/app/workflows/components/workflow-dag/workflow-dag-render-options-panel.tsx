import * as React from 'react';
import {WorkflowDagRenderOptions} from './workflow-dag';

export class WorkflowDagRenderOptionsPanel extends React.Component<WorkflowDagRenderOptions & {onChange: (changed: WorkflowDagRenderOptions) => void}> {
    private get workflowDagRenderOptions() {
        return this.props as WorkflowDagRenderOptions;
    }

    public render() {
        return (
            <>
                <a
                    onClick={() =>
                        this.props.onChange({
                            ...this.workflowDagRenderOptions,
                            showArtifacts: !this.workflowDagRenderOptions.showArtifacts
                        })
                    }
                    className={this.workflowDagRenderOptions.showArtifacts ? 'active' : ''}
                    title='Toggle artifacts'>
                    <i className='fa fa-file-alt' />
                </a>
                <a
                    onClick={() =>
                        this.props.onChange({
                            ...this.workflowDagRenderOptions,
                            expandNodes: new Set()
                        })
                    }
                    title='Collapse all nodes'>
                    <i className='fa fa-compress fa-fw' data-fa-transform='rotate-45' />
                </a>
                <a
                    onClick={() =>
                        this.props.onChange({
                            ...this.workflowDagRenderOptions,
                            expandNodes: new Set(['*'])
                        })
                    }
                    title='Expand all nodes'>
                    <i className='fa fa-expand fa-fw' data-fa-transform='rotate-45' />
                </a>
            </>
        );
    }
}
