import {Ticker} from 'argo-ui/src/index';
import * as React from 'react';
import {Link} from 'react-router-dom';
import {Workflow} from '../../../../models';
import {uiUrl} from '../../../shared/base';
import {PhaseIcon} from '../../../shared/components/phase-icon';
import {Timestamp} from '../../../shared/components/timestamp';
import {formatDuration, wfDuration} from '../../../shared/duration';
import {WorkflowDrawer} from '../workflow-drawer/workflow-drawer';

interface WorkflowsRowProps {
    workflow: Workflow;
    onChange: (key: string) => void;
    select: (wf: Workflow) => void;
}

interface WorkflowRowState {
    hideDrawer: boolean;
    selected: boolean;
}

export class WorkflowsRow extends React.Component<WorkflowsRowProps, WorkflowRowState> {
    constructor(props: WorkflowsRowProps) {
        super(props);
        this.state = {
            hideDrawer: true,
            selected: false
        };
    }

    public render() {
        const wf = this.props.workflow;
        return (
            <div className='workflows-list__row-container'>
                <div className='row argo-table-list__row'>
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
                                this.props.select(this.props.workflow);
                            }}
                        />
                        <PhaseIcon value={wf.status.phase} />
                    </div>
                    <Link to={uiUrl(`workflows/${wf.metadata.namespace}/${wf.metadata.name}`)} className='row small-11'>
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
                            name={wf.metadata.name}
                            namespace={wf.metadata.namespace}
                            onChange={key => {
                                this.props.onChange(key);
                            }}
                        />
                    )}
                </div>
            </div>
        );
    }
}
