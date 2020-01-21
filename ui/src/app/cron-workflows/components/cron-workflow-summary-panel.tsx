import * as React from 'react';

import {CronWorkflow} from '../../../models';
import {Timestamp} from '../../shared/components/timestamp';
import {YamlEditor} from '../../shared/components/yaml/yaml-editor';
import {services} from '../../shared/services';

const jsonMergePatch = require('json-merge-patch');

interface Props {
    cronWorkflow: CronWorkflow;
    onChange: (cronWorkflow: CronWorkflow) => void;

    onError(error: Error): void;
}

export const CronWorkflowSummaryPanel = (props: Props) => {
    const attributes = [
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
        {title: 'Suspended', value: props.cronWorkflow.spec.suspend},
        {title: 'Created', value: <Timestamp date={props.cronWorkflow.metadata.creationTimestamp} />}
    ];
    return (
        <div>
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

            <div className='white-box'>
                <div className='white-box__details'>
                    <YamlEditor
                        editing={false}
                        value={props.cronWorkflow}
                        onSubmit={(value: CronWorkflow) => {
                            // magic - we get the latest from the server and then apply the changes from the rendered version to this
                            const original = props.cronWorkflow;
                            const patch = jsonMergePatch.generate(original, value) || {};
                            services.cronWorkflows
                                .get(props.cronWorkflow.metadata.name, props.cronWorkflow.metadata.namespace)
                                .then(latest => jsonMergePatch.apply(latest, patch))
                                .then(patched => services.cronWorkflows.update(patched, props.cronWorkflow.metadata.name, props.cronWorkflow.metadata.namespace))
                                .then(updated => props.onChange(updated))
                                .catch(error => props.onError(error));
                        }}
                    />
                </div>
            </div>
        </div>
    );
};
