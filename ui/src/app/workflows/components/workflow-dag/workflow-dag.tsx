import * as classNames from 'classnames';
import * as dagre from 'dagre';
import * as React from 'react';

import * as models from '../../../../models';
import {NodePhase} from '../../../../models';
import {Utils} from '../../../shared/utils';
import {defaultNodesToDisplay} from '../workflow-details/workflow-details';

export const defaultWorkflowDagRenderOptions: WorkflowDagRenderOptions = {
    horizontal: false,
    zoom: 1,
    nodesToDisplay: defaultNodesToDisplay
};

export interface WorkflowDagRenderOptions {
    horizontal: boolean;
    zoom: number;
    nodesToDisplay: string[];
}

export interface WorkflowDagProps {
    workflow: models.Workflow;
    selectedNodeId?: string;
    nodeClicked?: (node: models.NodeStatus) => any;
    renderOptions: WorkflowDagRenderOptions;
}

require('./workflow-dag.scss');

export class WorkflowDag extends React.Component<WorkflowDagProps> {
    private get zoom() {
        return this.props.renderOptions.zoom || 1;
    }

    private get nodeSize() {
        return 32 / this.zoom;
    }
    private static iconPath(phase: NodePhase, suspended: boolean) {
        if (suspended) {
            return (
                <path
                    fill='currentColor'
                    d='M144 479H48c-26.5 0-48-21.5-48-48V79c0-26.5 21.5-48 48-48h96c26.5 0 48 21.5 48 48v352c0 26.5-21.5 48-48 48zm304-48V79c0-26.5-21.5-48-48-48h-96c-26.5 0-48 21.5-48 48v352c0 26.5 21.5 48 48 48h96c26.5 0 48-21.5 48-48z'
                />
            );
        }
        switch (phase) {
            case 'Pending':
                return (
                    <path
                        fill='currentColor'
                        d='M256,8C119,8,8,119,8,256S119,504,256,504,504,393,504,256,393,8,256,8Zm92.49,313h0l-20,25a16,16,0,0,1-22.49,2.5h0l-67-49.72a40,40,0,0,1-15-31.23V112a16,16,0,0,1,16-16h32a16,16,0,0,1,16,16V256l58,42.5A16,16,0,0,1,348.49,321Z'
                    />
                );
            case 'Failed':
            case 'Error':
                return (
                    <path
                        fill='currentColor'
                        d='M242.72 256l100.07-100.07c12.28-12.28 12.28-32.19 0-44.48l-22.24-22.24c-12.28-12.28-32.19-12.28-44.48 0L176 189.28 75.93 89.21c-12.28-12.28-32.19-12.28-44.48 0L9.21 111.45c-12.28 12.28-12.28 32.19 0 44.48L109.28 256 9.21 356.07c-12.28 12.28-12.28 32.19 0 44.48l22.24 22.24c12.28 12.28 32.2 12.28 44.48 0L176 322.72l100.07 100.07c12.28 12.28 32.2 12.28 44.48 0l22.24-22.24c12.28-12.28 12.28-32.19 0-44.48L242.72 256z'
                    />
                );
            case 'Skipped':
            case 'Succeeded':
                return (
                    <path
                        fill='currentColor'
                        d='M173.898 439.404l-166.4-166.4c-9.997-9.997-9.997-26.206 0-36.204l36.203-36.204c9.997-9.998 26.207-9.998 36.204 0L192 312.69 432.095 72.596c9.997-9.997 26.207-9.997 36.204 0l36.203 36.204c9.997 9.997 9.997 26.206 0 36.204l-294.4 294.401c-9.998 9.997-26.207 9.997-36.204-.001z'
                    />
                );
            case 'Running':
                return (
                    <path
                        fill='currentColor'
                        d='M288 39.056v16.659c0 10.804 7.281 20.159 17.686 23.066C383.204 100.434 440 171.518 440 256c0 101.689-82.295 184-184 184-101.689 0-184-82.295-184-184 0-84.47 56.786-155.564 134.312-177.219C216.719 75.874 224 66.517 224 55.712V39.064c0-15.709-14.834-27.153-30.046-23.234C86.603 43.482 7.394 141.206 8.003 257.332c.72 137.052 111.477 246.956 248.531 246.667C393.255 503.711 504 392.788 504 256c0-115.633-79.14-212.779-186.211-240.236C302.678 11.889 288 23.456 288 39.056z'
                    />
                );
        }
    }

    public render() {
        const graph = new dagre.graphlib.Graph();
        // https://github.com/dagrejs/dagre/wiki
        graph.setGraph({
            edgesep: this.nodeSize,
            nodesep: this.nodeSize * 2,
            rankdir: this.props.renderOptions.horizontal ? 'LR' : 'TB',
            ranksep: this.nodeSize * 2
        });
        graph.setDefaultEdgeLabel(() => ({}));
        const nodes = (this.props.workflow.status && this.props.workflow.status.nodes) || {};
        Object.values(nodes).forEach(node => {
            const label = Utils.shortNodeName(node);
            if (this.filterNode(node)) {
                graph.setNode(node.id, {label, width: 1, height: 1, ...nodes[node.id]});
            } else {
                graph.setNode(node.id, {label, width: this.nodeSize, height: this.nodeSize, ...nodes[node.id]});
            }
        });
        Object.keys(nodes).forEach(nodeId => {
            const node = nodes[nodeId];
            (node.children || []).forEach(childId => {
                // make sure workflow is in consistent state and child node exist
                if (nodes[childId]) {
                    graph.setEdge(nodeId, childId);
                }
            });
        });
        const onExitHandlerNodeId = Object.keys(nodes).find(id => nodes[id].name === `${this.props.workflow.metadata.name}.onExit`);
        if (onExitHandlerNodeId) {
            this.getOutboundNodes(this.props.workflow.metadata.name).forEach(nodeId => graph.setEdge(nodeId, onExitHandlerNodeId));
        }

        dagre.layout(graph);
        const size = this.getGraphSize(graph.nodes().map(id => graph.node(id)));
        return (
          <div className='workflow-dag' >
            <svg style={{width: size.width, height: size.height}}>
                <g transform={`translate(${this.nodeSize},${this.nodeSize})`}>
                    {graph
                        .edges()
                        .map(edge => graph.edge(edge))
                        .map(edge => edge.points.map((p, i) => `${i === 0 ? 'M' : 'L'} ${p.x} ${p.y}`).join(', '))
                        .map(points => (
                            <path key={`line/${points}`} d={points} className='line' />
                        ))}
                    {graph.nodes().map(id => {
                        const node = graph.node(id) as models.NodeStatus & dagre.Node;
                        return (
                            <g key={`node/${id}`} transform={`translate(${node.x},${node.y})`}>
                                <circle
                                    r={node.width / 2}
                                    className={classNames(
                                        'workflow-dag__node',
                                        'workflow-dag__node-status',
                                        'workflow-dag__node-status--' + (Utils.isNodeSuspended(node) ? 'suspended' : node.phase.toLocaleLowerCase()),
                                        {
                                            active: node.id === this.props.selectedNodeId
                                        }
                                    )}
                                    onClick={() => this.props.nodeClicked && this.props.nodeClicked(node)}
                                />
                                {!this.filterNode(node) && (
                                    <>
                                        {this.icon(node.phase, Utils.isNodeSuspended(node))}
                                        <g transform={`translate(0,${node.height})`}>
                                            <text textAnchor='middle' className='label'>{WorkflowDag.truncate(node.label)}</text>
                                        </g>
                                    </>
                                )}
                            </g>
                        );
                    })}
                </g>
            </svg>
          </div>
        );
    }

    private static truncate(label:string) {
        const number = 16;
        return label.length <= number ? label : label.substr(0, number-3)+"...";
    }

    private icon(phase: NodePhase, suspended: boolean) {
        return (
            <g>
                <g transform={`translate(-${this.nodeSize / 4},-${this.nodeSize / 4}), scale(${0.032 / this.zoom})`} color='white'>
                    {WorkflowDag.iconPath(phase, suspended)}
                </g>
                {phase === 'Running' && (
                    <animateTransform attributeType='xml' attributeName='transform' type='rotate' from='0 0 0 ' to='360 0 0' dur='1s' additive='sum' repeatCount='indefinite' />
                )}
            </g>
        );
    }

    private getOutboundNodes(nodeID: string): string[] {
        const node = this.props.workflow.status.nodes[nodeID];
        if (node.type === 'Pod' || node.type === 'Skipped') {
            return [node.id];
        }
        let outbound = Array<string>();
        for (const outboundNodeID of node.outboundNodes || []) {
            const outNode = this.props.workflow.status.nodes[outboundNodeID];
            if (outNode.type === 'Pod') {
                outbound.push(outboundNodeID);
            } else {
                outbound = outbound.concat(this.getOutboundNodes(outboundNodeID));
            }
        }
        return outbound;
    }

    private filterNode(node: models.NodeStatus) {
        // Filter the node if it is a virtual node or a Retry node with one child
        return (
            !(this.props.renderOptions.nodesToDisplay.includes('type:' + node.type) && this.props.renderOptions.nodesToDisplay.includes('phase:' + node.phase)) ||
            (node.type === 'Retry' && node.children.length === 1)
        );
    }

    private getGraphSize(nodes: dagre.Node[]): {width: number; height: number} {
        let width = 0;
        let height = 0;
        nodes.forEach(node => {
            width = Math.max(node.x + node.width , width);
            height = Math.max(node.y + node.height , height);
        });
        return {width: width + this.nodeSize * 2, height: height + this.nodeSize * 2};
    }
}
