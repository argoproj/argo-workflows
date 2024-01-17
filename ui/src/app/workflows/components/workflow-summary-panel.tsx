import {Ticker} from 'argo-ui';
import * as React from 'react';

import {labels, NODE_PHASE, Workflow} from '../../../models';
import {uiUrl} from '../../shared/base';
import {DurationPanel} from '../../shared/components/duration-panel';
import {Phase} from '../../shared/components/phase';
import {Timestamp} from '../../shared/components/timestamp';
import {ConditionsPanel} from '../../shared/conditions-panel';
import {Consumer} from '../../shared/context';
import {wfDuration} from '../../shared/duration';
import {ResourcesDuration} from '../../shared/resources-duration';
import {WorkflowCreatorInfo} from './workflow-creator-info/workflow-creator-info';
import {WorkflowFrom} from './workflow-from';
import {WorkflowLabels} from './workflow-labels/workflow-labels';

export const WorkflowSummaryPanel = (props: {workflow: Workflow}) => (
    <Ticker disabled={props.workflow && props.workflow.status.phase !== NODE_PHASE.RUNNING}>
        {() => {
            const attributes: {title: string; value: any}[] = [
                {title: 'Status', value: <Phase value={props.workflow.status.phase} />},
                {title: 'Message', value: props.workflow.status.message},
                {title: 'Name', value: props.workflow.metadata.name},
                {title: 'Namespace', value: props.workflow.metadata.namespace},
                {title: 'From', value: <WorkflowFrom namespace={props.workflow.metadata.namespace} labels={props.workflow.metadata.labels} />},
                {
                    title: 'Labels',
                    value: (
                        <Consumer>
                            {ctx => (
                                <WorkflowLabels
                                    workflow={props.workflow}
                                    onChange={(key, value) => ctx.navigation.goto(uiUrl(`workflows/${props.workflow.metadata.namespace}?label=${key}=${value}`))}
                                />
                            )}
                        </Consumer>
                    )
                },
                {title: 'Started', value: <Timestamp date={props.workflow.status.startedAt} />},
                {title: 'Finished ', value: <Timestamp date={props.workflow.status.finishedAt} />},
                {
                    title: 'Duration',
                    value: (
                        <DurationPanel
                            phase={props.workflow.status.phase}
                            duration={wfDuration(props.workflow.status)}
                            estimatedDuration={props.workflow.status.estimatedDuration}
                        />
                    )
                },
                {title: 'Progress', value: props.workflow.status.progress || '-'}
            ];
            const creator = props.workflow.metadata.labels[labels.creator];
            if (creator) {
                attributes.push({
                    title: 'Creator',
                    value: (
                        <Consumer>
                            {ctx => (
                                <WorkflowCreatorInfo
                                    workflow={props.workflow}
                                    onChange={(key, value) => ctx.navigation.goto(uiUrl(`workflows/${props.workflow.metadata.namespace}?label=${key}=${value}`))}
                                />
                            )}
                        </Consumer>
                    )
                });
            }
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
