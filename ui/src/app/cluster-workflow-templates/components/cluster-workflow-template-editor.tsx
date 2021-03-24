import * as React from 'react';

import {Tabs} from 'argo-ui';
import {WorkflowTemplate} from '../../../models';
import {LabelsAndAnnotationsEditor} from '../../shared/components/editors/labels-and-annotations-editor';
import {MetadataEditor} from '../../shared/components/editors/metadata-editor';
import {WorkflowParametersEditor} from '../../shared/components/editors/workflow-parameters-editor';
import {ObjectEditor} from '../../shared/components/object-editor/object-editor';

export const ClusterWorkflowTemplateEditor = ({
    onChange,
    template,
    onError,
    onTabSelected,
    selectedTabKey
}: {
    template: WorkflowTemplate;
    onChange: (template: WorkflowTemplate) => void;
    onError: (error: Error) => void;
    onTabSelected?: (tab: string) => void;
    selectedTabKey?: string;
}) => {
    return (
        <Tabs
            key='tabs'
            navTransparent={true}
            selectedTabKey={selectedTabKey}
            onTabSelected={onTabSelected}
            tabs={[
                {
                    key: 'manifest',
                    title: 'Manifest',
                    content: <ObjectEditor type='io.argoproj.workflow.v1alpha1.WorkflowTemplate' value={template} onChange={x => onChange({...x})} />
                },
                {
                    key: 'spec',
                    title: 'Spec',
                    content: <WorkflowParametersEditor value={template.spec} onChange={spec => onChange({...template, spec})} onError={onError} />
                },
                {
                    key: 'metadata',
                    title: 'MetaData',
                    content: <MetadataEditor value={template.metadata} onChange={metadata => onChange({...template, metadata})} />
                },
                {
                    key: 'workflow-metadata',
                    title: 'Workflow MetaData',
                    content: (
                        <LabelsAndAnnotationsEditor
                            value={template.spec.workflowMetadata}
                            onChange={workflowMetadata => onChange({...template, spec: {...template.spec, workflowMetadata}})}
                        />
                    )
                }
            ]}
        />
    );
};
