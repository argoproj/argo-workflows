import * as React from 'react';
import {Workflow} from '../../../../models';

import {Loading} from '../../../shared/components/loading';
import {ConditionsPanel} from '../../../shared/conditions-panel';
import {formatDuration} from '../../../shared/duration';
import {services} from '../../../shared/services';
import {WorkflowLabels} from '../workflow-labels/workflow-labels';

require('./workflow-drawer.scss');

interface WorkflowDrawerProps {
    name: string;
    namespace: string;
    onChange: (key: string) => void;
}

interface WorkflowDrawerState {
    workflow?: Workflow;
}

export class WorkflowDrawer extends React.Component<WorkflowDrawerProps, WorkflowDrawerState> {
    constructor(props: Readonly<WorkflowDrawerProps>) {
        super(props);
        this.state = {};
    }

    public componentDidMount() {
        services.workflows.get(this.props.namespace, this.props.name).then(workflow => {
            this.setState({workflow});
        });
    }

    public render() {
        if (!this.state.workflow) {
            return <Loading />;
        }
        const wf = this.state.workflow;
        return (
            <div className='workflow-drawer'>
                {!wf.status.message ? null : (
                    <div className='workflow-drawer__section workflow-drawer__message'>
                        <div className='workflow-drawer__title workflow-drawer__message--label'>MESSAGE</div>
                        <div className='workflow-drawer__message--content'>{wf.status.message}</div>
                    </div>
                )}
                {!wf.status.conditions ? null : (
                    <div className='workflow-drawer__section'>
                        <div className='workflow-drawer__title'>CONDITIONS</div>
                        <div className='workflow-drawer__conditions'>
                            <ConditionsPanel conditions={wf.status.conditions} />
                        </div>
                    </div>
                )}
                {!wf.status.resourcesDuration ? null : (
                    <div className='workflow-drawer__section'>
                        <div className='workflow-drawer__resourcesDuration'>
                            <div className='workflow-drawer__title'>
                                RESOURCES DURATION&nbsp;
                                <a href='https://github.com/argoproj/argo/blob/master/docs/resource-duration.md' target='_blank'>
                                    <i className='fas fa-info-circle' />
                                </a>
                            </div>
                            <div className='workflow-drawer__resourcesDuration--container'>
                                <div>
                                    <span className='workflow-drawer__resourcesDuration--value'>{formatDuration(wf.status.resourcesDuration.cpu, 1)}</span>
                                    <span className='workflow-drawer__resourcesDuration--label'>(*1 CPU)</span>
                                </div>
                                <div>
                                    <span className='workflow-drawer__resourcesDuration--value'>{formatDuration(wf.status.resourcesDuration.memory, 1)}</span>
                                    <span className='workflow-drawer__resourcesDuration--label'>(*100Mi Memory)</span>
                                </div>
                            </div>
                        </div>
                    </div>
                )}
                <div className='workflow-drawer__section workflow-drawer__labels'>
                    <div className='workflow-drawer__title'>LABELS</div>
                    <div className='workflow-drawer__labels--list'>
                        <WorkflowLabels
                            workflow={wf}
                            onChange={key => {
                                this.props.onChange(key);
                            }}
                        />
                    </div>
                </div>
            </div>
        );
    }
}
