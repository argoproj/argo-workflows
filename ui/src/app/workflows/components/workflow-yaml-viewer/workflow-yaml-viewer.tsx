import {SlideContents} from 'argo-ui';
import * as React from 'react';

import * as models from '../../../../models';
import {ObjectEditor} from '../../../shared/components/object-editor/object-editor';
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
        if (parentNode) {
            contents.push(
                <div key='parent-node'>
                    <h4>{normalizeNodeName(props.selectedNode.displayName || props.selectedNode.name)}</h4>
                    <ObjectEditor type='io.argoproj.workflow.v1alpha1.Template' value={getResolvedTemplates(props.workflow, parentNode)} />
                </div>
            );
        }

        const currentNodeTemplate = getResolvedTemplates(props.workflow, props.selectedNode);
        if (currentNodeTemplate) {
            contents.push(
                <div key='current-node'>
                    <h4>{props.selectedNode.name}</h4>
                    <ObjectEditor type='io.argoproj.workflow.v1alpha1.Template' value={currentNodeTemplate} />
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
                contents={<ObjectEditor type='io.argoproj.workflow.v1alpha1.Template' value={templates} />}
                className='workflow-yaml-section'
            />
        );
    }

    const storedTemplates = props.workflow.status.storedTemplates;
    if (storedTemplates && Object.keys(storedTemplates).length) {
        contents.push(
            <SlideContents
                title='Stored Templates'
                key='stored-templates'
                contents={<ObjectEditor type='io.argoproj.workflow.v1alpha1.Template' value={storedTemplates} />}
                className='workflow-yaml-section'
            />
        );
    }

    return <div className='workflow-yaml-viewer'>{contents}</div>;
}
