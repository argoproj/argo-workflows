import {Ticker} from 'argo-ui/src/components/ticker';
import * as React from 'react';
import {useState} from 'react';
import {Link} from 'react-router-dom';

import * as models from '../../../../models';
import {isArchivedWorkflow, Workflow} from '../../../../models';
import {ANNOTATION_DESCRIPTION, ANNOTATION_TITLE} from '../../../shared/annotations';
import {uiUrl} from '../../../shared/base';
import {Loading} from '../../../shared/components/loading';
import {DurationPanel} from '../../../shared/components/duration-panel';
import {PhaseIcon} from '../../../shared/components/phase-icon';
import {Timestamp} from '../../../shared/components/timestamp';
import {wfDuration} from '../../../shared/duration';
import {WorkflowDrawer} from '../workflow-drawer/workflow-drawer';

require('./workflows-row.scss');

interface WorkflowsRowProps {
    workflow: Workflow;
    onChange: (key: string) => void;
    select: (wf: Workflow) => void;
    checked: boolean;
    columns: models.Column[];
}

export function WorkflowsRow(props: WorkflowsRowProps) {
    const [hideDrawer, setHideDrawer] = useState(true);
    const wf = props.workflow;
    // title + description vars
    const title = wf.metadata.annotations?.[ANNOTATION_TITLE] ?? wf.metadata.name;
    const description = (wf.metadata.annotations?.[ANNOTATION_DESCRIPTION] && `\n${wf.metadata.annotations[ANNOTATION_DESCRIPTION]}`) || '';
    const hasAnnotation = title !== wf.metadata.name && description !== '';
    const markdown = `${title}${description}`;

    return (
        <div className='workflows-list__row-container'>
            <div className='row argo-table-list__row'>
                <div className='columns small-1 workflows-list__status'>
                    <input
                        type='checkbox'
                        className='workflows-list__status--checkbox'
                        checked={props.checked}
                        onClick={e => {
                            e.stopPropagation();
                        }}
                        onChange={() => {
                            props.select(props.workflow);
                        }}
                    />
                    <PhaseIcon value={wf.status.phase} />
                </div>
                <div className='small-11 row'>
                    <Link
                        to={{
                            pathname: uiUrl(`workflows/${wf.metadata.namespace}/${wf.metadata.name}`),
                            search: `?uid=${wf.metadata.uid}`
                        }}
                        className='columns small-2'>
                        <div className='wf-rows-name'>{hasAnnotation ? <SuspenseReactMarkdownGfm markdown={markdown} /> : markdown}</div>
                    </Link>
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
                    {/* CSS has text-overflow, but sometimes it's still too long for the column for some reason, so slice it too. 180 chars are not visible on a 4k screen */}
                    <div className='columns small-2'>{wf.status.message?.slice(0, 180) || '-'}</div>
                    <div className='columns small-1'>
                        <div className='workflows-list__labels-container'>
                            <div
                                onClick={e => {
                                    e.preventDefault();
                                    setHideDrawer(!hideDrawer);
                                }}
                                className={`workflows-row__action workflows-row__action--${hideDrawer ? 'show' : 'hide'}`}>
                                {hideDrawer ? (
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
                    {(props.columns || []).map(column => {
                        // best not to make any assumptions and wait until this data is filled
                        const value = (column.type === 'label' ? wf?.metadata?.labels?.[column.key] : wf?.metadata?.annotations?.[column.key]) ?? 'unknown';
                        return (
                            <div key={column.name} className='columns small-1'>
                                {value}
                            </div>
                        );
                    })}
                    {hideDrawer ? <span /> : <WorkflowDrawer name={wf.metadata.name} namespace={wf.metadata.namespace} onChange={props.onChange} />}
                </div>
            </div>
        </div>
    );
}

// lazy load ReactMarkdown (and remark-gfm) as it is a large optional component (which can be split into a separate bundle)
const LazyReactMarkdownGfm = React.lazy(() => {
    return import(/* webpackChunkName: "react-markdown-plus-gfm" */ './react-markdown-gfm');
});

function SuspenseReactMarkdownGfm(props: {markdown: string}) {
    return (
        <React.Suspense fallback={<Loading />}>
            <LazyReactMarkdownGfm markdown={props.markdown} />
        </React.Suspense>
    );
}
