import {Ticker} from 'argo-ui/src/index';
import * as React from 'react';
import {Link} from 'react-router-dom';
import * as models from '../../../../models';
import {uiUrl} from '../../../shared/base';
import {PhaseIcon} from '../../../shared/components/phase-icon';
import {Timestamp} from '../../../shared/components/timestamp';
import {formatDuration, wfDuration} from '../../../shared/duration';
import {services} from '../../../shared/services';
import {WorkflowDrawer} from '../workflow-drawer/workflow-drawer';

interface WorkflowsRowProps {
    workflow: models.Workflow;
    onChange: (key: string) => void;
    select: (wf: models.Workflow) => void;
}

interface WorkflowRowState {
    hideDrawer: boolean;
    workflow: models.Workflow;
    selected: boolean;
}

export class WorkflowsRow extends React.Component<WorkflowsRowProps, WorkflowRowState> {
    constructor(props: WorkflowsRowProps) {
        super(props);
        this.state = {
            workflow: this.props.workflow,
            hideDrawer: true,
            selected: false
        };
    }

    public render() {
        const wf = this.state.workflow;
        return (
            <div className='workflows-list__row-container'>
                <Link className='row argo-table-list__row' to={uiUrl(`workflows/${wf.metadata.namespace}/${wf.metadata.name}`)}>
                    <div className='columns small-1 workflows-list__status'>
                        <input
                            type='checkbox'
                            className='workflows-list__status--checkbox'
                            checked={this.state.selected}
                            onClick={e => {
                                e.stopPropagation();
                            }}
                            onChange={e => {
                                this.setState({selected: !this.state.selected});
                                this.props.select(this.state.workflow);
                            }}
                        />
                        <PhaseIcon value={wf.status.phase} />
                    </div>
                    <div className='columns small-3'>{wf.metadata.name}</div>
                    <div className='columns small-2'>{wf.metadata.namespace}</div>
                    <div className='columns small-2'>
                        <Timestamp date={wf.status.startedAt} />
                    </div>
                    <div className='columns small-2'>
                        <Timestamp date={wf.status.finishedAt} />
                    </div>
                    <div className='columns small-1'>
                        <Ticker>{() => formatDuration(wfDuration(wf.status))}</Ticker>
                    </div>
                    <div className='columns small-1'>
                        <div className='workflows-list__labels-container'>
                            <div
                                onClick={e => {
                                    e.preventDefault();
                                    this.fetchFullWorkflow();
                                    this.setState({hideDrawer: !this.state.hideDrawer});
                                }}
                                className={`workflows-row__action workflows-row__action--${this.state.hideDrawer ? 'show' : 'hide'}`}>
                                {this.state.hideDrawer ? (
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
                    </div>
                </Link>
                {this.state.hideDrawer ? (
                    <span />
                ) : (
                    <WorkflowDrawer
                        workflow={wf}
                        onChange={key => {
                            this.props.onChange(key);
                        }}
                    />
                )}
            </div>
        );
    }

    private fetchFullWorkflow(): void {
        services.workflows.get(this.props.workflow.metadata.namespace, this.props.workflow.metadata.name).then(wf => {
            this.setState({workflow: wf});
        });
    }
}
