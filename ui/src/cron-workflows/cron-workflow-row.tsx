import {Ticker} from 'argo-ui/src/index';
import * as React from 'react';
import {Link} from 'react-router-dom';

import {ANNOTATION_DESCRIPTION, ANNOTATION_TITLE} from '../shared/annotations';
import {uiUrl} from '../shared/base';
import {ErrorIcon} from '../shared/components/fa-icons';
import {SuspenseReactMarkdownGfm} from '../shared/components/suspense-react-markdown-gfm';
import {Timestamp} from '../shared/components/timestamp';
import {getErrorCondition} from '../shared/conditions-panel';
import {getNextScheduledTime, getResolvedSchedules} from '../shared/cron';
import {CronWorkflow} from '../shared/models';
import {escapeInvalidMarkdown} from '../workflows/utils';
import {PrettySchedule} from './pretty-schedule';

require('./cron-workflow-row.scss');

interface CronWorkflowRowProps {
    workflow: CronWorkflow;
    displayISOFormatCreation: boolean;
    displayISOFormatNextScheduled: boolean;
}

export function CronWorkflowRow(props: CronWorkflowRowProps) {
    const wf = props.workflow;
    // title + description vars
    const title = (wf.metadata.annotations?.[ANNOTATION_TITLE] && `${escapeInvalidMarkdown(wf.metadata.annotations[ANNOTATION_TITLE])}`) ?? wf.metadata.name;
    const description = (wf.metadata.annotations?.[ANNOTATION_DESCRIPTION] && `\n${escapeInvalidMarkdown(wf.metadata.annotations[ANNOTATION_DESCRIPTION])}`) || '';
    const markdown = `${title}${description}`;
    const errorCondition = getErrorCondition(wf.status?.conditions || []);
    // the schedules are shown as configured, but described and counted down with `H` resolved
    const resolvedSchedules = getResolvedSchedules(wf);

    return (
        <div className='cron-workflows-list__row-container'>
            <div className='row argo-table-list__row'>
                <div className='columns small-1'>
                    {errorCondition ? (
                        <span title={errorCondition.message}>
                            <ErrorIcon />
                        </span>
                    ) : wf.spec.suspend ? (
                        <i className='fa fa-pause' />
                    ) : (
                        <i className='fa fa-clock' />
                    )}
                </div>
                <Link to={{pathname: uiUrl(`cron-workflows/${wf.metadata.namespace}/${wf.metadata.name}`)}} className='columns small-2'>
                    <div className={description.length ? 'wf-rows-name' : ''} aria-valuetext={markdown}>
                        <SuspenseReactMarkdownGfm markdown={markdown} />
                    </div>
                </Link>
                <div className='columns small-1'>{wf.metadata.namespace}</div>
                <div className='columns small-1'>{wf.spec.timezone}</div>
                <div className='columns small-2'>
                    {wf.spec.schedules.map(schedule => (
                        <>
                            {schedule}
                            <br />
                        </>
                    ))}
                </div>
                <div className='columns small-1'>
                    {resolvedSchedules.map(schedule => (
                        <>
                            <PrettySchedule schedule={schedule} />
                            <br />
                        </>
                    ))}
                </div>
                <div className='columns small-2'>
                    <Timestamp date={wf.metadata.creationTimestamp} displayISOFormat={props.displayISOFormatCreation} />
                </div>
                <div className='columns small-2'>
                    {wf.spec.suspend ? (
                        ''
                    ) : (
                        <Ticker intervalMs={1000}>
                            {() => <Timestamp date={getCronNextScheduledTime(resolvedSchedules, wf.spec.timezone)} displayISOFormat={props.displayISOFormatNextScheduled} />}
                        </Ticker>
                    )}
                </div>
            </div>
        </div>
    );
}

function getCronNextScheduledTime(schedules: string[], timezone: string): Date {
    let out: Date;
    schedules.forEach(schedule => {
        const next = getNextScheduledTime(schedule, timezone);
        // a schedule which is not resolved yet, or which we cannot parse, has no next time
        if (next && (!out || next.getTime() < out.getTime())) {
            out = next;
        }
    });
    return out;
}
