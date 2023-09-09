import * as React from 'react';

import {Tabs} from 'argo-ui';
import {Workflow} from '../../../models';
import {MetadataEditor} from '../../shared/components/editors/metadata-editor';
import {WorkflowParametersEditor} from '../../shared/components/editors/workflow-parameters-editor';
import {ObjectEditor} from '../../shared/components/object-editor/object-editor';

export function WorkflowEditor({
    selectedTabKey,
    onTabSelected,
    onError,
    onChange,
    template
}: {
    template: Workflow;
    onChange: (template: Workflow) => void;
    onError: (error: Error) => void;
    onTabSelected?: (tab: string) => void;
    selectedTabKey?: string;
}) {
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
                    content: <ObjectEditor type='io.argoproj.workflow.v1alpha1.Workflow' value={template} onChange={x => onChange({...x})} />
                },
                {
                    key: 'parameters',
                    title: 'Parameters',
                    content: <WorkflowParametersEditor value={template.spec} onChange={spec => onChange({...template, spec})} onError={onError} />
                },
                {
                    key: 'metadata',
                    title: 'MetaData',
                    content: <MetadataEditor value={template.metadata} onChange={metadata => onChange({...template, metadata})} />
                }
            ]}
        />
    );
}
