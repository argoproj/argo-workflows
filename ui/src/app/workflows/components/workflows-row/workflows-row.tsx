import {Ticker} from 'argo-ui/src/index';
import * as React from 'react';
import {Link} from 'react-router-dom';
import * as models from '../../../../models';
import {uiUrl} from '../../../shared/base';
import {PhaseIcon} from '../../../shared/components/phase-icon';
import {Timestamp} from '../../../shared/components/timestamp';
import {formatDuration, wfDuration} from '../../../shared/duration';
import {WorkflowDrawer} from '../workflow-drawer/workflow-drawer';

interface WorkflowsRowProps {
    workflow: models.Workflow;
    onChange: (key: string) => void;
}

interface WorkflowRowState {
    hideDrawer: boolean;
}

export class WorkflowsRow extends React.Component<WorkflowsRowProps, WorkflowRowState> {
    constructor(props: WorkflowsRowProps) {
        super(props);
        this.state = {
            hideDrawer: true
        };
    }

    public render() {
        const wf = this.props.workflow;
        return (
            <div className='workflows-list__row-container'>
                <Link className='row argo-table-list__row' to={uiUrl(`workflows/${wf.metadata.namespace}/${wf.metadata.name}`)}>
                    <div className='columns small-1 workflows-list__status'>
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
        );
    }
}
