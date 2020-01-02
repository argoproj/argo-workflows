import * as React from 'react';

import {WorkflowTemplate} from '../../../models';
import {YamlEditor} from '../../shared/components/yaml-editor/yaml-editor';
import {services} from '../../shared/services';

const jsonMergePatch = require('json-merge-patch');

export const WorkflowTemplateSummaryPanel = (props: {workflowTemplate: WorkflowTemplate}) => {
    const attributes = [
        {title: 'Name', value: props.workflowTemplate.metadata.name},
        {title: 'Namespace', value: props.workflowTemplate.metadata.namespace},
        {title: 'Created At', value: props.workflowTemplate.metadata.creationTimestamp}
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
                        input={props.workflowTemplate}
                        onSave={wfTmpl => {
                            const patch = jsonMergePatch.generate(props.workflowTemplate, wfTmpl);

                            const spec = JSON.parse(JSON.stringify(props.workflowTemplate));
                            return services.workflowTemplate.update(
                                jsonMergePatch.apply(spec, JSON.parse(patch)),
                                props.workflowTemplate.metadata.name,
                                props.workflowTemplate.metadata.namespace,
                            );
                        }}
                    />
                </div>
            </div>
        </div>
    );
};