import {Ticker} from 'argo-ui/src/index';
import * as React from 'react';
import {Link} from 'react-router-dom';
import * as models from '../../../../models';
import {isArchivedWorkflow, Workflow} from '../../../../models';
import {ANNOTATION_DESCRIPTION, ANNOTATION_TITLE} from '../../../shared/annotations';
import {uiUrl} from '../../../shared/base';
import {DurationPanel} from '../../../shared/components/duration-panel';
import {PhaseIcon} from '../../../shared/components/phase-icon';
import {Timestamp} from '../../../shared/components/timestamp';
import {wfDuration} from '../../../shared/duration';
import {WorkflowDrawer} from '../workflow-drawer/workflow-drawer';

interface WorkflowsRowProps {
    workflow: Workflow;
    onChange: (key: string) => void;
    select: (wf: Workflow) => void;
    checked: boolean;
    columns: models.Column[];
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
                <div className='row argo-table-list__row'>
                    <div className='columns small-1 workflows-list__status'>
                        <input
                            type='checkbox'
                            className='workflows-list__status--checkbox'
                            checked={this.props.checked}
                            onClick={e => {
                                e.stopPropagation();
                            }}
                            onChange={e => {
                                this.props.select(this.props.workflow);
                            }}
                        />
                        <PhaseIcon value={wf.status.phase} />
                    </div>
                    <Link
                        to={{
                            pathname: uiUrl(`workflows/${wf.metadata.namespace}/${wf.metadata.name}`),
                            search: `?uid=${wf.metadata.uid}`
                        }}
                        className='small-11 row'>
                        <div className='columns small-2'>
                            {(wf.metadata.annotations && wf.metadata.annotations[ANNOTATION_TITLE]) || wf.metadata.name}
                            {wf.metadata.annotations && wf.metadata.annotations[ANNOTATION_DESCRIPTION] ? <p>{wf.metadata.annotations[ANNOTATION_DESCRIPTION]}</p> : null}
                        </div>
                        <div className='columns small-1'>{wf.metadata.namespace}</div>
                        <div className='columns small-1'>
                            <Timestamp date={wf.status.startedAt} />
                        </div>
                        <div className='columns small-1'>
                            <Timestamp date={wf.status.finishedAt} />
                        </div>
                        <div className='columns small-1'>
                            <Ticker>{() => <DurationPanel phase={wf.status.phase} duration={wfDuration(wf.status)} estimatedDuration={wf.status.estimatedDuration} />}</Ticker>
                        </div>
                        <div className='columns small-1'>{wf.status.progress || '-'}</div>
                        <div className='columns small-2'>{wf.status.message || '-'}</div>
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
                        <div className='columns small-1'>{isArchivedWorkflow(wf) ? 'true' : 'false'}</div>
                        {(this.props.columns || []).map(column => {
                            const value = wf.metadata?.labels[column.key];
                            return (
                                <div key={column.name} className='columns small-1'>
                                    {value}
                                </div>
                            );
                        })}
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
                    </Link>
                </div>
            </div>
        );
    }
}
