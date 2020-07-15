import * as React from 'react';
import * as models from '../../../../models';

require('./workflow-labels.scss');

interface WorkflowLabelsProps {
    workflow: models.Workflow;
    onChange: (key: string) => void;
}

export class WorkflowLabels extends React.Component<WorkflowLabelsProps, {}> {
    constructor(props: WorkflowLabelsProps) {
        super(props);
    }

    public render() {
        const labels = [];
        const w = this.props.workflow;
        if (w.metadata.labels) {
            labels.push(
                Object.keys(w.metadata.labels).map(key => (
                    <div
                        className='tag'
                        key={`${w.metadata.uid}-${key}`}
                        onClick={async e => {
                            e.preventDefault();
                            this.props.onChange(key);
                        }}>
                        <div className='key'>{key}</div>
                        <div className='value'>{w.metadata.labels[key]}</div>
                    </div>
                ))
            );
        } else {
            labels.push(<div key={`${w.metadata.uid}-none`}> No labels </div>);
        }

        return <div className='wf-row-labels'>{labels}</div>;
    }
}
