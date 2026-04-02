import {Tabs} from 'argo-ui/src/components/tabs/tabs';
import * as React from 'react';

import {GraphViewer} from '../../shared/components/editors/graph-viewer';
import {MetadataEditor} from '../../shared/components/editors/metadata-editor';
import {WorkflowParametersEditor} from '../../shared/components/editors/workflow-parameters-editor';
import {ObjectEditor} from '../../shared/components/object-editor';
import type {Lang} from '../../shared/components/object-parser';
import {Workflow} from '../../shared/models';

export function WorkflowEditor({
    selectedTabKey,
    onTabSelected,
    onError,
    onChange,
    onLangChange,
    workflow,
    serialization,
    lang
}: {
    workflow: Workflow;
    serialization: string;
    lang: Lang;
    onChange: (workflow: string | Workflow) => void;
    onError: (error: Error) => void;
    onTabSelected?: (tab: string) => void;
    onLangChange: (lang: Lang) => void;
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
                    content: (
                        <ObjectEditor
                            type='io.argoproj.workflow.v1alpha1.Workflow'
                            value={workflow}
                            text={serialization}
                            lang={lang}
                            onLangChange={onLangChange}
                            onChange={onChange}
                        />
                    )
                },
                {
                    key: 'parameters',
                    title: 'Parameters',
                    content: <WorkflowParametersEditor value={workflow.spec} onChange={spec => onChange({...workflow, spec})} onError={onError} />
                },
                {
                    key: 'metadata',
                    title: 'MetaData',
                    content: <MetadataEditor value={workflow.metadata} onChange={metadata => onChange({...workflow, metadata})} />
                },
                {
                    key: 'graph',
                    title: 'Graph',
                    content: <GraphViewer workflowDefinition={workflow} />
                }
            ]}
        />
    );
}
