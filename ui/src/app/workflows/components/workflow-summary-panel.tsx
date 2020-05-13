import {Ticker} from 'argo-ui';
import * as React from 'react';

import {NODE_PHASE, Workflow} from '../../../models';
import {ConditionsPanel} from '../../shared/conditions-panel';
import {formatDuration, wfDuration} from '../../shared/duration';
import {ResourcesDuration} from '../../shared/resources-duration';

export const WorkflowSummaryPanel = (props: {workflow: Workflow}) => (
    <Ticker disabled={props.workflow && props.workflow.status.phase !== NODE_PHASE.RUNNING}>
        {() => {
            const attributes: {title: string; value: any}[] = [
                {title: 'Status', value: props.workflow.status.phase},
                {title: 'Message', value: props.workflow.status.message},
                {title: 'Name', value: props.workflow.metadata.name},
                {title: 'Namespace', value: props.workflow.metadata.namespace},
                {title: 'Started At', value: props.workflow.status.startedAt},
                {title: 'Finished At', value: props.workflow.status.finishedAt || '-'},
                {title: 'Duration', value: formatDuration(wfDuration(props.workflow.status))}
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
