import * as React from 'react';
import * as models from '../../../../models';

require('./workflow-labels.scss');

interface WorkflowLabelsProps {
    workflow: models.Workflow;
    onChange: (key: string, value: string) => void;
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
                Object.entries(w.metadata.labels).map(([key, value]) => (
                    <div
                        title={`List workflows labelled with ${key}=${value}`}
                        className='tag'
                        key={`${w.metadata.uid}-${key}`}
                        onClick={async e => {
                            e.preventDefault();
                            this.props.onChange(key, value);
                        }}>
                        <div className='key'>{key}</div>
                        <div className='value'>{value}</div>
                    </div>
                ))
            );
        } else {
            labels.push(<div key={`${w.metadata.uid}-none`}> No labels </div>);
        }

        return <div className='wf-row-labels'>{labels}</div>;
    }
}
