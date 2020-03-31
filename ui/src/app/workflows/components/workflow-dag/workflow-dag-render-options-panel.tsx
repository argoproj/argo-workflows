import * as classNames from 'classnames';
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
                    className={classNames({active: this.props.horizontal})}
                    onClick={() =>
                        this.props.onChange({
                            ...this.workflowDagRenderOptions,
                            horizontal: !this.props.horizontal
                        })
                    }
                    title='Horizontal layout'>
                    <i className='fa fa-project-diagram' />
                </a>
                <a
                    className={classNames({active: this.props.zoom > 1})}
                    onClick={() =>
                        this.props.onChange({
                            ...this.workflowDagRenderOptions,
                            zoom: this.props.zoom === 1 ? 2 : 1
                        })
                    }
                    title='Zoom into the timeline'>
                    2x
                </a>
            </>
        );
    }
}
