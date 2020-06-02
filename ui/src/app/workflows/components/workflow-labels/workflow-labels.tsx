import * as React from 'react';
import * as models from '../../../../models';

require('./workflow-labels.scss');

interface WorkflowLabelsProps {
    workflow: models.Workflow;
    onChange: (key: string) => void;
}

export class WorkflowLabels extends React.Component<WorkflowLabelsProps, {hidden: boolean}> {
    constructor(props: WorkflowLabelsProps) {
        super(props);
        this.state = {hidden: true};
    }

    public render() {
        const labels = [];
        if (!this.state.hidden) {
            const w = this.props.workflow;
            if (w.metadata.labels) {
                labels.push(
                    Object.keys(w.metadata.labels).map(key => (
                        <div
                            className='tag'
                            key={`${w.metadata.namespace}-${w.metadata.name}-${key}`}
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
                labels.push(<div key={`${w.metadata.namespace}-${w.metadata.name}-none`}> No labels </div>);
            }
        }

        return (
            <div className='wf-row-labels'>
                {labels}
                <div
                    onClick={e => {
                        e.preventDefault();
                        this.setState({hidden: !this.state.hidden});
                    }}
                    className={`wf-row-labels__action wf-row-labels__action--${this.state.hidden ? 'show' : 'hide'}`}>
                    {this.state.hidden ? (
                        <span>
                            SHOW <i className='fas fa-caret-down' />{' '}
                        </span>
                    ) : (
                        <span>
                            HIDE <i className='fas fa-caret-up' />
                        </span>
                    )}
                </div>
            </div>
        );
    }
}
