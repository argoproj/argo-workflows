import * as React from 'react';

import {CronWorkflow} from '../../../models';
import {Timestamp} from '../../shared/components/timestamp';
import {YamlEditor} from '../../shared/components/yaml-editor/yaml-editor';
import {services} from '../../shared/services';

const jsonMergePatch = require('json-merge-patch');

export const CronWorkflowSummaryPanel = (props: {cronWf: CronWorkflow}) => {
    const attributes = [
        {title: 'Name', value: props.cronWf.metadata.name},
        {title: 'Namespace', value: props.cronWf.metadata.namespace},
        {title: 'Schedule', value: props.cronWf.spec.schedule},
        {title: 'Concurrency Policy', value: props.cronWf.spec.concurrencyPolicy ? props.cronWf.spec.concurrencyPolicy : 'Allow'},
        {title: 'Starting Deadline Seconds', value: props.cronWf.spec.startingDeadlineSeconds},
        {title: 'Successful Jobs History Limit', value: props.cronWf.spec.successfulJobsHistoryLimit},
        {title: 'Failed Jobs History Limit', value: props.cronWf.spec.failedJobsHistoryLimit},
        {title: 'Suspended', value: props.cronWf.spec.suspend},
        {title: 'Created', value: <Timestamp date={props.cronWf.metadata.creationTimestamp} />}
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
                        minHeight={800}
                        submitMode={false}
                        input={props.cronWf}
                        onSave={cronWf => {
                            const patch = jsonMergePatch.generate(props.cronWf, cronWf);

                            const spec = JSON.parse(JSON.stringify(props.cronWf));
                            return services.cronWorkflows.update(jsonMergePatch.apply(spec, JSON.parse(patch)), props.cronWf.metadata.name, props.cronWf.metadata.namespace);
                        }}
                    />
                </div>
            </div>
        </div>
    );
};
