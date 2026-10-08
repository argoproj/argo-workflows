import {SlideContents} from 'argo-ui/src/components/slide-contents/slide-contents';
import * as React from 'react';

import {SerializingObjectEditor} from '../../../shared/components/object-editor';
import {PhaseIcon} from '../../../shared/components/phase-icon';
import * as models from '../../../shared/models';
import {getResolvedTemplates} from '../../../shared/template-resolution';

interface WorkflowYamlViewerProps {
    workflow: models.Workflow;
    selectedNode: models.NodeStatus;
}

function normalizeNodeName(name: string) {
    const parts = name.replace(/([(][^)]*[)])/g, '').split('.');
    return parts[parts.length - 1];
}

export function WorkflowYamlViewer(props: WorkflowYamlViewerProps) {
    const contents: JSX.Element[] = [];
    contents.push(<h3 key='title'>Node</h3>);

    if (props.selectedNode) {
        const parentNode = props.workflow.status.nodes[props.selectedNode.boundaryID];
        let renderedNode = false;
        if (parentNode) {
            const parentNodeTemplate = getResolvedTemplates(props.workflow, parentNode);
            if (parentNodeTemplate) {
                renderedNode = true;
                contents.push(
                    <div key='parent-node'>
                        <h4>{normalizeNodeName(props.selectedNode.displayName || props.selectedNode.name)}</h4>
                        <SerializingObjectEditor type='io.argoproj.workflow.v1alpha1.Template' value={parentNodeTemplate} />
                    </div>
                );
            }
        }

        const currentNodeTemplate = getResolvedTemplates(props.workflow, props.selectedNode);
        if (currentNodeTemplate) {
            renderedNode = true;
            contents.push(
                <div key='current-node'>
                    <h4>{props.selectedNode.name}</h4>
                    <SerializingObjectEditor type='io.argoproj.workflow.v1alpha1.Template' value={currentNodeTemplate} />
                </div>
            );
        }

        if (!renderedNode) {
            // The node's template could not be resolved. This happens for archived
            // workflows whose offloaded template rows were garbage-collected after
            // completion — the template is genuinely unrecoverable. Show a clear note
            // rather than a silently empty (or crashing) panel.
            contents.push(
                <div key='no-template' className='argo-field' style={{marginTop: '1em'}}>
                    <PhaseIcon value='Error' />
                    Template not available: the template for this node could not be resolved. It may have been garbage-collected after the workflow was archived.
                </div>
            );
        }
    }

    const templates = props.workflow.spec.templates;
    if (templates && Object.keys(templates).length) {
        contents.push(
            <SlideContents
                title='Templates'
                key='templates'
                contents={<SerializingObjectEditor type='io.argoproj.workflow.v1alpha1.Template' value={templates} />}
                className='workflow-yaml-section'
            />
        );
    }

    // Show the storedTemplates section when it has content and is not just the hydrated copy
    // of spec.templates. Offloaded workflows carry the marker, and hydration fills the spec;
    // templateRef workflows have no marker and hold their resolved templates only here.
    const isTemplatesOffloaded = props.workflow.status.storedTemplateSpecs?.uid != null;
    const storedTemplates = props.workflow.status.storedTemplates;
    if (storedTemplates && Object.keys(storedTemplates).length && (!isTemplatesOffloaded || !templates || Object.keys(templates).length === 0)) {
        contents.push(
            <SlideContents
                title='Stored Templates'
                key='stored-templates'
                contents={<SerializingObjectEditor type='io.argoproj.workflow.v1alpha1.Template' value={storedTemplates} />}
                className='workflow-yaml-section'
            />
        );
    }

    return <div className='workflow-yaml-viewer'>{contents}</div>;
}
