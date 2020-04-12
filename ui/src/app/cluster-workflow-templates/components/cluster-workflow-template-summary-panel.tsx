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

export const ClusterWorkflowTemplateSummaryPanel = (props: Props) => {
    const attributes = [
        {title: 'Name', value: props.template.metadata.name},
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
                            return services.clusterWorkflowTemplate
                                .update(value, props.template.metadata.name)
                                .then(clusterWorkflowTemplate => props.onChange(clusterWorkflowTemplate))
                                .catch(err => props.onError(err));
                        }}
                    />
                </div>
            </div>
        </div>
    );
};
