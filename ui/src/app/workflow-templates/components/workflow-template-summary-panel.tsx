import * as React from 'react';

import {WorkflowTemplate} from '../../../models';
import {YamlEditor} from "../../shared/components/yaml-editor/yaml-editor";
import {services} from "../../shared/services";

export const WorkflowTemplateSummaryPanel = (props: { workflowTemplate: WorkflowTemplate }) => {
    const attributes = [
        {title: 'Name', value: props.workflowTemplate.metadata.name},
        {title: 'Namespace', value: props.workflowTemplate.metadata.namespace},
        {title: 'Created At', value: props.workflowTemplate.metadata.creationTimestamp},
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
                            services.workflowTemplate
                                .update(JSON.parse(wfTmpl), props.workflowTemplate.metadata.name, props.workflowTemplate.metadata.namespace)
                                .then();
                        }}
                    />
                </div>
            </div>
        </div>
    );
};
