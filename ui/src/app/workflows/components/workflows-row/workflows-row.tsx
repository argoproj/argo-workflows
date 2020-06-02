import {Ticker} from 'argo-ui/src/index';
import * as React from 'react';
import {Link} from 'react-router-dom';
import * as models from '../../../../models';
import {uiUrl} from '../../../shared/base';
import {PhaseIcon} from '../../../shared/components/phase-icon';
import {Timestamp} from '../../../shared/components/timestamp';
import {formatDuration, wfDuration} from '../../../shared/duration';
import {WorkflowLabels} from '../workflow-labels/workflow-labels';

interface WorkflowsRowProps {
    workflow: models.Workflow;
    onChange: (key: string) => void;
}
export class WorkflowsRow extends React.Component<WorkflowsRowProps, {hideLabels: boolean}> {
    constructor(props: WorkflowsRowProps) {
        super(props);
        this.state = {hideLabels: true};
    }

    public render() {
        const wf = this.props.workflow;
        return (
            <div>
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
                                    this.setState({hideLabels: !this.state.hideLabels});
                                }}
                                className={`workflows-row__action workflows-row__action--${this.state.hideLabels ? 'show' : 'hide'}`}>
                                {this.state.hideLabels ? (
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
                {this.state.hideLabels ? (
                    <span />
                ) : (
                    <div>
                        <WorkflowLabels
                            workflow={wf}
                            onChange={key => {
                                this.props.onChange(key);
                            }}
                        />
                    </div>
                )}
            </div>
        );
    }
}
