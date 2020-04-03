import {Duration, Ticker} from 'argo-ui';
import * as moment from 'moment';
import * as React from 'react';

import {NODE_PHASE, Workflow} from '../../../models';
import {ConditionsPanel} from '../../shared/conditions-panel';
import {ResourcesDuration} from '../../shared/resources-duration';
import {services} from "../../shared/services";

export const WorkflowSummaryPanel = (props: {workflow: Workflow}) => (
    <Ticker disabled={props.workflow && props.workflow.status.phase !== NODE_PHASE.RUNNING}>
        {now => {
            const endTime = props.workflow.status.finishedAt ? moment(props.workflow.status.finishedAt) : now;
            const duration = endTime.diff(moment(props.workflow.status.startedAt)) / 1000;

            const attributes = [
                {title: 'Status', value: props.workflow.status.phase},
                {title: 'Message', value: props.workflow.status.message},
                {title: 'Name', value: props.workflow.metadata.name},
                {title: 'Namespace', value: props.workflow.metadata.namespace},
                {title: 'Started At', value: props.workflow.status.startedAt},
                {title: 'Finished At', value: props.workflow.status.finishedAt || '-'},
                {title: 'Duration', value: <Duration durationMs={duration} />},
                {title: 'Logs', value: <a href={services.workflows.getLogsUrl(props.workflow)}>Download all logs as .tgz</a>}
            ];
            if (props.workflow.status.resourcesDuration) {
                attributes.push({
                    title: 'Resources Duration',
                    value: <ResourcesDuration resourcesDuration={props.workflow.status.resourcesDuration} />
                });
            }
            if (props.workflow.status.conditions) {
                attributes.push({
                    title: 'Conditions',
                    value: <ConditionsPanel conditions={props.workflow.status.conditions} />
                });
            }
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
