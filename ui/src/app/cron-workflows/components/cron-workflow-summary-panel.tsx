import * as React from 'react';

import * as kubernetes from 'argo-ui/src/models/kubernetes';
import {CronWorkflow} from '../../../models';
import {ResourceEditor} from '../../shared/components/resource-editor/resource-editor';
import {Timestamp} from '../../shared/components/timestamp';
import {ConditionsPanel} from '../../shared/conditions-panel';
import {services} from '../../shared/services';
import {WorkflowLink} from '../../workflows/components/workflow-link';

const jsonMergePatch = require('json-merge-patch');
const parser = require('cron-parser');

interface Props {
    cronWorkflow: CronWorkflow;
    onChange: (cronWorkflow: CronWorkflow) => void;
}

export const CronWorkflowSummaryPanel = (props: Props) => {
    const specAttributes = [
        {title: 'Name', value: props.cronWorkflow.metadata.name},
        {title: 'Namespace', value: props.cronWorkflow.metadata.namespace},
        {title: 'Schedule', value: props.cronWorkflow.spec.schedule},
        {title: 'Timezone', value: props.cronWorkflow.spec.timezone},
        {
            title: 'Concurrency Policy',
            value: props.cronWorkflow.spec.concurrencyPolicy ? props.cronWorkflow.spec.concurrencyPolicy : 'Allow'
        },
        {title: 'Starting Deadline Seconds', value: props.cronWorkflow.spec.startingDeadlineSeconds},
        {title: 'Successful Jobs History Limit', value: props.cronWorkflow.spec.successfulJobsHistoryLimit},
        {title: 'Failed Jobs History Limit', value: props.cronWorkflow.spec.failedJobsHistoryLimit},
        {title: 'Suspended', value: (!!props.cronWorkflow.spec.suspend).toString()},
        {title: 'Created', value: <Timestamp date={props.cronWorkflow.metadata.creationTimestamp} />}
    ];
    const statusAttributes = [
        {title: 'Active', value: props.cronWorkflow.status.active ? getCronWorkflowActiveWorkflowList(props.cronWorkflow.status.active) : <i>No Workflows Active</i>},
        {title: 'Last Scheduled Time', value: props.cronWorkflow.status.lastScheduledTime},
        {
            title: 'Next Scheduled Time',
            value: getNextScheduledTime(props.cronWorkflow.spec.schedule, props.cronWorkflow.spec.timezone) + ' (assumes workflow-controller is in UTC)'
        },
        {title: 'Conditions', value: <ConditionsPanel conditions={props.cronWorkflow.status.conditions} />}
    ];
    return (
        <div>
            <div className='white-box'>
                <div className='white-box__details'>
                    {specAttributes.map(attr => (
                        <div className='row white-box__details-row' key={attr.title}>
                            <div className='columns small-3'>{attr.title}</div>
                            <div className='columns small-9'>{attr.value}</div>
                        </div>
                    ))}
                </div>
            </div>

            <div className='white-box'>
                <div className='white-box__details'>
                    {statusAttributes.map(attr => (
                        <div className='row white-box__details-row' key={attr.title}>
                            <div className='columns small-3'>{attr.title}</div>
                            <div className='columns small-9'>{attr.value}</div>
                        </div>
                    ))}
                </div>
            </div>

            <div className='white-box'>
                <div className='white-box__details'>
                    <ResourceEditor
                        kind='CronWorkflow'
                        value={props.cronWorkflow}
                        onSubmit={(value: CronWorkflow) => {
                            // magic - we get the latest from the server and then apply the changes from the rendered version to this
                            const original = props.cronWorkflow;
                            const patch = jsonMergePatch.generate(original, value) || {};
                            return services.cronWorkflows
                                .get(props.cronWorkflow.metadata.name, props.cronWorkflow.metadata.namespace)
                                .then(latest => jsonMergePatch.apply(latest, patch))
                                .then(patched => services.cronWorkflows.update(patched, props.cronWorkflow.metadata.name, props.cronWorkflow.metadata.namespace))
                                .then(updated => props.onChange(updated));
                        }}
                    />
                </div>
            </div>
        </div>
    );
};

function getCronWorkflowActiveWorkflowList(active: kubernetes.ObjectReference[]) {
    return active.reverse().map(activeWf => <WorkflowLink key={activeWf.uid} namespace={activeWf.namespace} name={activeWf.name} />);
}

function getNextScheduledTime(schedule: string, tz: string): string {
    let out = '';
    try {
        out = parser
            .parseExpression(schedule, {utc: !tz, tz})
            .next()
            .toISOString();
    } catch (e) {
        // Do nothing
    }
    return out;
}
