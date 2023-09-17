import {ObjectMeta} from 'argo-ui/src/models/kubernetes';
import * as React from 'react';
import {WorkflowStatus} from '../../../../models';
import {Notice} from '../../../shared/components/notice';
import {Phase} from '../../../shared/components/phase';
import {WorkflowDag} from '../workflow-dag/workflow-dag';

interface Props {
    workflowMetadata: ObjectMeta;
    workflowStatus: WorkflowStatus;
    selectedNodeId: string;
    nodeClicked: (nodeId: string) => void;
}

export function WorkflowPanel(props: Props) {
    if (!props.workflowStatus.nodes && props.workflowStatus.phase) {
        return (
            <div className='argo-container'>
                <Notice>
                    <Phase value={props.workflowStatus.phase} />: {props.workflowStatus.message}
                </Notice>
            </div>
        );
    }

    return (
        <WorkflowDag
            workflowName={props.workflowMetadata.name}
            nodes={props.workflowStatus.nodes}
            artifactRepositoryRef={props.workflowStatus.artifactRepositoryRef}
            selectedNodeId={props.selectedNodeId}
            nodeClicked={props.nodeClicked}
        />
    );
}
