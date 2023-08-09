import * as React from 'react';
import {Workflow} from '../../../../models';

import {InlineTable} from '../../../shared/components/inline-table/inline-table';
import {Loading} from '../../../shared/components/loading';
import {ConditionsPanel} from '../../../shared/conditions-panel';
import {formatDuration} from '../../../shared/duration';
import {services} from '../../../shared/services';
import {WorkflowCreatorInfo} from '../workflow-creator-info/workflow-creator-info';
import {WorkflowFrom} from '../workflow-from';
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
        const wf = this.state.workflow;

        if (!wf) {
            return <Loading />;
        }

        return (
            <div className='workflow-drawer'>
                {!wf.status || !wf.status.message ? null : (
                    <div className='workflow-drawer__section workflow-drawer__message'>
                        <div className='workflow-drawer__title workflow-drawer__message--label'>MESSAGE</div>
                        <div className='workflow-drawer__message--content'>{wf.status.message}</div>
                    </div>
                )}
                {!wf.status || !wf.status.conditions ? null : (
                    <div className='workflow-drawer__section'>
                        <div className='workflow-drawer__title'>CONDITIONS</div>
                        <div className='workflow-drawer__conditions'>
                            <ConditionsPanel conditions={wf.status.conditions} />
                        </div>
                    </div>
                )}
                {!wf.status || !wf.status.resourcesDuration ? null : (
                    <div className='workflow-drawer__section'>
                        <div>
                            <InlineTable
                                rows={[
                                    {
                                        left: (
                                            <div className='workflow-drawer__title'>
                                                RESOURCES DURATION&nbsp;
                                                <a
                                                    href='https://github.com/argoproj/argo-workflows/blob/master/docs/resource-duration.md'
                                                    onClick={e => e.stopPropagation()}
                                                    target='_blank'>
                                                    <i className='fas fa-info-circle' />
                                                </a>
                                            </div>
                                        ),
                                        right: (
                                            <div>
                                                <div>
                                                    <span className='workflow-drawer__resourcesDuration--value'>{formatDuration(wf.status.resourcesDuration.cpu, 1)}</span>
                                                    <span>(*1 CPU)</span>
                                                </div>
                                                <div>
                                                    <span className='workflow-drawer__resourcesDuration--value'>{formatDuration(wf.status.resourcesDuration.memory, 1)}</span>
                                                    <span>(*100Mi Memory)</span>
                                                </div>
                                            </div>
                                        )
                                    }
                                ]}
                            />
                        </div>
                    </div>
                )}
                <div className='workflow-drawer__section'>
                    <div className='workflow-drawer__title'>FROM</div>
                    <div className='workflow-drawer__workflowFrom'>
                        <WorkflowFrom namespace={wf.metadata.namespace || 'default'} labels={wf.metadata.labels || {}} />
                    </div>
                </div>
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
                <div className='workflow-drawer__section workflow-drawer__labels'>
                    <div className='workflow-drawer__title'>Creator</div>
                    <div className='workflow-drawer__labels--list'>
                        <WorkflowCreatorInfo
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
