import * as React from 'react';

import {WorkflowTemplate} from '../../../models';
import {Timestamp} from '../../shared/components/timestamp';
import {YamlEditor} from '../../shared/components/yaml/yaml-editor';
import {services} from '../../shared/services';

interface Props {
    template: WorkflowTemplate;
    onChange: (template: WorkflowTemplate) => void;
    onError: (error: Error) => void;
}

export const WorkflowTemplateSummaryPanel = (props: Props) => {
    const attributes = [
        {title: 'Name', value: props.template.metadata.name},
        {title: 'Namespace', value: props.template.metadata.namespace},
        {title: 'Created', value: <Timestamp date={props.template.metadata.creationTimestamp} />}
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
                        value={props.template}
                        onSubmit={(value: WorkflowTemplate) => {
                            return services.workflowTemplate
                                .update(value, props.template.metadata.name, props.template.metadata.namespace)
                                .then(workflowTemplate => props.onChange(workflowTemplate))
                                .catch(err => props.onError(err));
                        }}
                    />
                </div>
            </div>
        </div>
    );
};
