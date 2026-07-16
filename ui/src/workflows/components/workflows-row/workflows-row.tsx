import {Ticker} from 'argo-ui/src/components/ticker';
import * as React from 'react';
import {useState} from 'react';
import {Link} from 'react-router-dom';

import {ANNOTATION_DESCRIPTION, ANNOTATION_TITLE} from '../../../shared/annotations';
import {DurationPanel} from '../../../shared/components/duration-panel';
import {Phase} from '../../../shared/components/phase';
import {Timestamp} from '../../../shared/components/timestamp';
import {useContext} from '../../../shared/context';
import * as models from '../../../shared/models';
import {Workflow} from '../../../shared/models';
import {WorkflowDrawer} from '../workflow-drawer/workflow-drawer';

import './workflows-row.scss';

interface WorkflowsRowProps {
    workflow: models.Workflow;
    onChange: (...args: any[]) => void;
    checked: boolean;
    onCheck?: (...args: any[]) => void;
    columns: models.Column[] | string[];
    select: (...args: any[]) => void;
    displayISOFormatStart: boolean;
    displayISOFormatFinished: boolean;
}

function escapeInvalidMarkdown(text: string): string {
    return text.replace(/([\\`*_{}[\]()#+\-.!])/g, '\\$1');
}

export function WorkflowsRow(props: WorkflowsRowProps) {
    const {workflow: wf} = props;
    const {navigation} = useContext();
    const [isOpen, setIsOpen] = useState(false);

    const hasAnnotation = !!(wf.metadata.annotations?.[ANNOTATION_DESCRIPTION] || wf.metadata.annotations?.[ANNOTATION_TITLE]);
    const description = (wf.metadata.annotations?.[ANNOTATION_DESCRIPTION] && `\n${escapeInvalidMarkdown(wf.metadata.annotations[ANNOTATION_DESCRIPTION])}`) || '';

    const renderColumns = () => {
        return props.columns.map((column: any) => {
            const col = typeof column === 'string' ? column : column?.name;
            if (col === 'name') {
                return (
                    <div key={col} className='columns workflows-list__col-name' onClick={e => e.stopPropagation()}>
                        <input type='checkbox' checked={props.checked} onChange={props.onCheck} style={{marginRight: '10px'}} />
                        <Link to={navigation.apis.workflows.getLink(wf.metadata.namespace, wf.metadata.name)}>
                            {wf.metadata.name}
                        </Link>
                        {hasAnnotation && (
                            <span className='workflows-list__has-annotation' title={description}>
                                <i className='fa fa-comment-alt' style={{marginLeft: '10px', color: '#6d7f8b'}} />
                            </span>
                        )}
                    </div>
                );
            }
            if (col === 'namespace') {
                return (
                    <div key={col} className='columns workflows-list__col-namespace'>
                        {wf.metadata.namespace}
                    </div>
                );
            }
            if (col === 'phase') {
                return (
                    <div key={col} className='columns workflows-list__col-phase'>
                        <Phase value={wf.status?.phase} />
                    </div>
                );
            }
            if (col === 'started') {
                return (
                    <div key={col} className='columns workflows-list__col-started'>
                        <Timestamp date={wf.status?.startedAt} displayISOFormat={props.displayISOFormatStart} />
                    </div>
                );
            }
            if (col === 'finished') {
                return (
                    <div key={col} className='columns workflows-list__col-finished'>
                        <Timestamp date={wf.status?.finishedAt} displayISOFormat={props.displayISOFormatFinished} />
                    </div>
                );
            }
            if (col === 'duration') {
                return (
                    <div key={col} className='columns workflows-list__col-duration'>
                        <Ticker>
                            {now => {
                                const end = wf.status?.finishedAt ? new Date(wf.status.finishedAt) : now;
                                const duration = wf.status?.startedAt ? (end.getTime() - new Date(wf.status.startedAt).getTime()) / 1000 : 0;
                                return <DurationPanel duration={duration} phase={wf.status?.phase} />;
                            }}
                        </Ticker>
                    </div>
                );
            }
            if (col === 'progress') {
                return (
                    <div key={col} className='columns workflows-list__col-progress'>
                        {wf.status?.progress || '-'}
                    </div>
                );
            }
            return null;
        });
    };

    return (
        <div className='workflows-row' onClick={props.select}>
            <div className='row workflows-list__row-content'>{renderColumns()}</div>
            {hasAnnotation && (
                <div onClick={e => e.stopPropagation()}>
                    <WorkflowDrawer
                        description={wf.metadata.annotations?.[ANNOTATION_DESCRIPTION]}
                        hasAnnotation={hasAnnotation}
                        name={wf.metadata.name}
                        namespace={wf.metadata.namespace}
                        onChange={props.onChange}
                        title={wf.metadata.annotations?.[ANNOTATION_TITLE]}
                    />
                </div>
            )}
        </div>
    );
}
