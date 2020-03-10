import * as classNames from 'classnames';
import * as dagre from 'dagre';
import * as React from 'react';

import * as models from '../../../../models';
import {NODE_PHASE} from '../../../../models';
import {Utils} from '../../../shared/utils';

export const defaultWorkflowDagRenderOptions: WorkflowDagRenderOptions = {
    hideSucceeded: false,
    horizontal: false,
    zoom: 1
};

export interface WorkflowDagRenderOptions {
    horizontal: boolean;
    zoom: number;
    hideSucceeded: boolean;
}

export interface WorkflowDagProps {
    workflow: models.Workflow;
    selectedNodeId?: string;
    nodeClicked?: (node: models.NodeStatus) => any;
    renderOptions: WorkflowDagRenderOptions;
}

interface Line {
    x1: number;
    y1: number;
    x2: number;
    y2: number;
    noArrow: boolean;
}

require('./workflow-dag.scss');

export class WorkflowDag extends React.Component<WorkflowDagProps> {
    private get zoom() {
        return this.props.renderOptions.zoom || 1;
    }

    private get nodeWidth() {
        return 32 / this.zoom;
    }

    private get nodeHeight() {
        return 32 / this.zoom;
    }

    public render() {
        const graph = new dagre.graphlib.Graph();
        // https://github.com/dagrejs/dagre/wiki
        graph.setGraph({
            edgesep: 20 / this.zoom,
            nodesep: 50 / this.zoom,
            rankdir: this.props.renderOptions.horizontal ? 'LR' : 'TB',
            ranksep: 50 / this.zoom
        });
        graph.setDefaultEdgeLabel(() => ({}));
        const nodes = (this.props.workflow.status && this.props.workflow.status.nodes) || {};
        Object.values(nodes).forEach(node => {
            const label = Utils.shortNodeName(node);
            if (this.isSmall(node)) {
                graph.setNode(node.id, {label, width: 1, height: 1, ...nodes[node.id]});
            } else {
                graph.setNode(node.id, {label, width: this.nodeWidth, height: this.nodeHeight, ...nodes[node.id]});
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
        const edges: {from: string; to: string; lines: Line[]}[] = [];
        graph.edges().forEach(edgeInfo => {
            const edge = graph.edge(edgeInfo);
            const lines: Line[] = [];
            if (edge.points.length > 1) {
                for (let i = 1; i < edge.points.length; i++) {
                    const toNode = nodes[edgeInfo.w];
                    lines.push({
                        x1: edge.points[i - 1].x,
                        y1: edge.points[i - 1].y,
                        x2: edge.points[i].x,
                        y2: edge.points[i].y,
                        noArrow: this.isSmall(toNode)
                    });
                }
            }
            edges.push({from: edgeInfo.v, to: edgeInfo.w, lines});
        });
        const size = this.getGraphSize(graph.nodes().map(id => graph.node(id)));
        return (
            <div className='workflow-dag' style={{width: size.width, height: size.height}}>
                {graph.nodes().map(id => {
                    const node = graph.node(id) as models.NodeStatus & dagre.Node;
                    const small = this.isSmall(node);
                    return (
                        <>
                            <div
                                key={id}
                                title={node.label}
                                className={classNames(
                                    'workflow-dag__node',
                                    `fas`,
                                    'workflow-dag__node-status',
                                    'workflow-dag__node-status--' + (Utils.isNodeSuspended(node) ? 'suspended' : node.phase.toLocaleLowerCase()),
                                    {
                                        active: node.id === this.props.selectedNodeId,
                                        virtual: this.filterNode(node),
                                        small
                                    }
                                )}
                                style={{
                                    left: node.x - node.width / 2,
                                    top: node.y - node.height / 2,
                                    width: node.width,
                                    height: node.height,
                                    lineHeight: this.nodeHeight + 'px',
                                    fontSize: 1 / this.zoom + 'em',
                                    borderRadius: this.nodeWidth / 2 + 'px'
                                }}
                                onClick={() => this.props.nodeClicked && this.props.nodeClicked(node)}
                            />
                            {!small && (
                                <div
                                    className='workflow-dag__node-title'
                                    style={{
                                        position: 'absolute',
                                        left: node.x - node.width,
                                        top: node.y + node.height / 2,
                                        width: node.width * 2,
                                        lineHeight: this.nodeHeight + 'px',
                                        textAlign: 'center',
                                        fontSize: 0.75 / this.zoom + 'em'
                                    }}>
                                    {node.label}
                                </div>
                            )}
                        </>
                    );
                })}
                {edges.map(edge => (
                    <div key={`${edge.from}-${edge.to}`} className='workflow-dag__edge'>
                        {edge.lines.map((line, i) => {
                            const distance = Math.sqrt(Math.pow(line.x1 - line.x2, 2) + Math.pow(line.y1 - line.y2, 2));
                            const xMid = (line.x1 + line.x2) / 2;
                            const yMid = (line.y1 + line.y2) / 2;
                            const angle = (Math.atan2(line.y1 - line.y2, line.x1 - line.x2) * 180) / Math.PI;
                            return (
                                <div
                                    className={classNames('workflow-dag__line', {'workflow-dag__line--no-arrow': line.noArrow})}
                                    key={i}
                                    style={{
                                        width: distance,
                                        left: xMid - distance / 2,
                                        top: yMid,
                                        transform: ` rotate(${angle}deg)`
                                    }}
                                />
                            );
                        })}
                    </div>
                ))}
            </div>
        );
    }

    private isSmall(node: models.NodeStatus) {
        return this.filterNode(node) || (this.props.renderOptions.hideSucceeded && node.phase === NODE_PHASE.SUCCEEDED);
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

    private isVirtual(node: models.NodeStatus) {
        return (node.type === 'StepGroup' || node.type === 'DAG' || node.type === 'TaskGroup') && !!node.boundaryID;
    }

    private filterNode(node: models.NodeStatus) {
        // Filter the node if it is a virtual node or a Retry node with one child
        return this.isVirtual(node) || (node.type === 'Retry' && node.children.length === 1);
    }

    private getGraphSize(nodes: dagre.Node[]): {width: number; height: number} {
        let width = 0;
        let height = 0;
        nodes.forEach(node => {
            width = Math.max(node.x + node.width / 2, width);
            height = Math.max(node.y + node.height / 2, height);
        });
        return {width, height};
    }
}
