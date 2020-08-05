import {Ticker} from 'argo-ui';
import * as React from 'react';

import {labels, NODE_PHASE, Workflow} from '../../../models';
import {ClusterWorkflowTemplateLink} from '../../cluster-workflow-templates/components/cluster-workflow-template-link';
import {CronWorkflowLink} from '../../cron-workflows/components/cron-workflow-link';
import {uiUrl} from '../../shared/base';
import {Phase} from '../../shared/components/phase';
import {Timestamp} from '../../shared/components/timestamp';
import {ConditionsPanel} from '../../shared/conditions-panel';
import {Consumer} from '../../shared/context';
import {formatDuration, wfDuration} from '../../shared/duration';
import {ResourcesDuration} from '../../shared/resources-duration';
import {WorkflowTemplateLink} from '../../workflow-templates/components/workflow-template-link';
import {WorkflowLabels} from './workflow-labels/workflow-labels';

export const WorkflowSummaryPanel = (props: {workflow: Workflow}) => (
    <Ticker disabled={props.workflow && props.workflow.status.phase !== NODE_PHASE.RUNNING}>
        {() => {
            const attributes: {title: string; value: any}[] = [
                {title: 'Status', value: <Phase value={props.workflow.status.phase} />},
                {title: 'Message', value: props.workflow.status.message},
                {title: 'Name', value: props.workflow.metadata.name},
                {title: 'Namespace', value: props.workflow.metadata.namespace}
            ];
            const workflowTemplate = props.workflow.metadata.labels[labels.workflowTemplate];
            if (workflowTemplate) {
                attributes.push({title: 'Workflow Template', value: <WorkflowTemplateLink namespace={props.workflow.metadata.namespace} name={workflowTemplate} />});
            }
            const clusterWorkflowTemplate = props.workflow.metadata.labels[labels.clusterWorkflowTemplate];
            if (clusterWorkflowTemplate) {
                attributes.push({title: 'Cluster Workflow Template', value: <ClusterWorkflowTemplateLink name={clusterWorkflowTemplate} />});
            }
            const cronWorkflow = props.workflow.metadata.labels[labels.cronWorkflow];
            if (cronWorkflow) {
                attributes.push({title: 'Cron Workflow', value: <CronWorkflowLink namespace={props.workflow.metadata.namespace} name={cronWorkflow} />});
            }
            attributes.push({
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
            });
            attributes.push({title: 'Started', value: <Timestamp date={props.workflow.status.startedAt} />});
            attributes.push({title: 'Finished ', value: <Timestamp date={props.workflow.status.finishedAt} />});
            attributes.push({title: 'Duration', value: formatDuration(wfDuration(props.workflow.status))});
            const creator = props.workflow.metadata.labels[labels.creator];
            if (creator) {
                attributes.push({title: 'Creator', value: creator});
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
