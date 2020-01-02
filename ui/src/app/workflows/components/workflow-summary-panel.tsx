import {Duration, Ticker} from 'argo-ui';
import * as moment from 'moment';
import * as React from 'react';

import {NODE_PHASE, Workflow} from '../../../models';

export const WorkflowSummaryPanel = (props: {workflow: Workflow}) => (
    <Ticker disabled={props.workflow && props.workflow.status.phase !== NODE_PHASE.RUNNING}>
        {now => {
            const endTime = props.workflow.status.finishedAt ? moment(props.workflow.status.finishedAt) : now;
            const duration = endTime.diff(moment(props.workflow.status.startedAt)) / 1000;

            const attributes = [
                {title: 'Status', value: props.workflow.status.phase},
                {title: 'Name', value: props.workflow.metadata.name},
                {title: 'Namespace', value: props.workflow.metadata.namespace},
                {title: 'Started At', value: props.workflow.status.startedAt},
                {title: 'Finished At', value: props.workflow.status.finishedAt || '-'},
                {title: 'Duration', value: <Duration durationMs={duration} />}
            ];
            return (
                <div className='white-box'>
                    <div className='white-box__details'>
                        {attributes.map(attr => (
                            <div className='row white-box__details-row' key={attr.title}>
                                <div className='columns small-3'>{attr.title}</div>
                                <div className='columns small-9'>{attr.value}</div>
                            </div>
                        ))}
                    </div>
                </div>
            );
        }}
    </Ticker>
);
