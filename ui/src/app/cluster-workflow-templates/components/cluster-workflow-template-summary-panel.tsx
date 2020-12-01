import * as React from 'react';

import {WorkflowTemplate} from '../../../models';
import {ExampleManifests} from '../../shared/components/example-manifests';
import {ResourceEditor} from '../../shared/components/resource-editor/resource-editor';
import {Timestamp} from '../../shared/components/timestamp';
import {services} from '../../shared/services';

interface Props {
    template: WorkflowTemplate;
    onChange: (template: WorkflowTemplate) => void;
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
                    <ResourceEditor
                        kind='ClusterWorkflowTemplate'
                        title='Update Cluster Workflow Template'
                        value={props.template}
                        onSubmit={(value: WorkflowTemplate) =>
                            services.clusterWorkflowTemplate.update(value, props.template.metadata.name).then(clusterWorkflowTemplate => props.onChange(clusterWorkflowTemplate))
                        }
                    />
                    <p>
                        <ExampleManifests />
                    </p>
                </div>
            </div>
        </div>
    );
};
