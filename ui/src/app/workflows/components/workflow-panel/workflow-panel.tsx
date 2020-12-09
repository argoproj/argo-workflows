import {ObjectMeta} from 'argo-ui/src/models/kubernetes';
import * as React from 'react';
import {WorkflowDag} from '..';
import {WorkflowStatus} from '../../../../models';
import {Notice} from '../../../shared/components/notice';
import {Phase} from '../../../shared/components/phase';

interface Props {
    workflowMetadata: ObjectMeta;
    workflowStatus: WorkflowStatus;
    selectedNodeId: string;
    nodeClicked: (nodedId: string) => void;
}

export class WorkflowPanel extends React.Component<Props> {
    public render() {
        if (!this.props.workflowStatus.nodes && this.props.workflowStatus.phase) {
            return (
                <div className='argo-container'>
                    <Notice>
                        <Phase value={this.props.workflowStatus.phase} />: {this.props.workflowStatus.message}
                    </Notice>
                </div>
            );
        }

        return (
            <WorkflowDag
                workflowName={this.props.workflowMetadata.name}
                nodes={this.props.workflowStatus.nodes}
                selectedNodeId={this.props.selectedNodeId}
                nodeClicked={this.props.nodeClicked}
            />
        );
    }
}
