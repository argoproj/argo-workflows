import * as React from 'react';

import {Tabs} from 'argo-ui';
import {WorkflowTemplate} from '../../../models';
import {MetadataEditor} from '../../shared/components/editors/metadata-editor';
import {WorkflowSpecEditor} from '../../shared/components/editors/workflow-spec-editor';
import {ObjectEditor} from '../../shared/components/resource-editor/object-editor';

export const WorkflowTemplateSummaryPanel = (props: {template: WorkflowTemplate; onChange: (template: WorkflowTemplate) => void; onError: (error: Error) => void}) => {
    return (
        <Tabs
            key='tabs'
            navTransparent={true}
            tabs={[
                {
                    key: 'visual',
                    title: 'Visual',
                    content: (
                        <>
                            <MetadataEditor value={props.template.metadata} onChange={metadata => props.onChange({...props.template, metadata})} />
                            <WorkflowSpecEditor value={props.template.spec} onChange={spec => props.onChange({...props.template, spec})} onError={error => props.onError(error)} />
                        </>
                    )
                },
                {
                    key: 'manifest',
                    title: 'Manifest',
                    content: (
                        <ObjectEditor
                            type='io.argoproj.workflow.v1alpha1.WorkflowTemplate'
                            value={props.template}
                            onChange={template => props.onChange({...template})}
                            onError={error => props.onError(error)}
                        />
                    )
                }
            ]}
        />
    );
};
