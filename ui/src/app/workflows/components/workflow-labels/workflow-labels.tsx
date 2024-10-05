import * as React from 'react';

import * as models from '../../../../models';

import './workflow-labels.scss';

interface WorkflowLabelsProps {
    workflow: models.Workflow;
    onChange: (key: string, value: string) => void;
}

export function WorkflowLabels(props: WorkflowLabelsProps) {
    const labels = [];
    const w = props.workflow;
    if (w.metadata.labels) {
        labels.push(
            Object.entries(w.metadata.labels).map(([key, value]) => (
                <div
                    title={`List workflows labelled with ${key}=${value}`}
                    className='tag'
                    key={`${w.metadata.uid}-${key}`}
                    onClick={async e => {
                        e.preventDefault();
                        props.onChange(key, value);
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
