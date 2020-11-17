import * as React from 'react';

import {Tabs} from 'argo-ui';
import {WorkflowTemplate} from '../../../models';
import {MetadataEditor} from '../../shared/components/editors/metadata-editor';
import {WorkflowSpecEditor} from '../../shared/components/editors/workflow-spec-editor';
import {ObjectEditor} from '../../shared/components/object-editor/object-editor';

export const WorkflowTemplateEditor = (props: {
    template: WorkflowTemplate;
    onChange: (template: WorkflowTemplate) => void;
    onError: (error: Error) => void;
    onTabSelected: (tab: string) => void;
    selectedTabKey: string;
}) => {
    return (
        <Tabs
            key='tabs'
            navTransparent={true}
            selectedTabKey={props.selectedTabKey}
            onTabSelected={props.onTabSelected}
            tabs={[
                {
                    key: 'spec',
                    title: 'Spec',
                    content: <WorkflowSpecEditor value={props.template.spec} onChange={spec => props.onChange({...props.template, spec})} onError={props.onError} />
                },
                {
                    key: 'metadata',
                    title: 'MetaData',
                    content: <MetadataEditor value={props.template.metadata} onChange={metadata => props.onChange({...props.template, metadata})} />
                },
                {
                    key: 'manifest',
                    title: 'Manifest',
                    content: (
                        <ObjectEditor
                            type='io.argoproj.workflow.v1alpha1.WorkflowTemplate'
                            value={props.template}
                            onChange={template => props.onChange({...template})}
                            onError={props.onError}
                        />
                    )
                }
            ]}
        />
    );
};
