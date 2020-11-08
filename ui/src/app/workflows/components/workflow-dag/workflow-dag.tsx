import {Ticker} from 'argo-ui';
import * as React from 'react';

import {NODE_PHASE, NodePhase, NodeStatus} from '../../../../models';
import {GraphPanel} from '../../../shared/components/graph/graph-panel';
import {Graph} from '../../../shared/components/graph/types';
import {Loading} from '../../../shared/components/loading';
import {Utils} from '../../../shared/utils';
import {getCollapsedNodeName, getMessage, getNodeParent, isCollapsedNode} from './graph/collapsible-node';
import {icons} from './icons';
import {WorkflowDagRenderOptionsPanel} from './workflow-dag-render-options-panel';

export interface WorkflowDagRenderOptions {
    expandNodes: Set<string>;
}

export interface WorkflowDagProps {
    workflowName: string;
    nodes: {[nodeId: string]: NodeStatus};
    selectedNodeId?: string;
    nodeClicked?: (nodeId: string) => any;
}

const types = new Set(['Pod', 'Steps', 'DAG', 'Retry', 'Skipped', 'Suspend', 'TaskGroup', 'StepGroup']);

function dagPhase(n: NodeStatus) {
    return n.type === 'Suspend' && n.phase === 'Running' ? 'Suspended' : n.phase;
}

function nodeLabel(n: NodeStatus) {
    const p = dagPhase(n);
    return {
        label: Utils.shortNodeName(n),
        type: n.type,
        icon: icons[p],
        classNames: p
    };
}

export class WorkflowDag extends React.Component<WorkflowDagProps, WorkflowDagRenderOptions> {
    /**
     * Return and SVG path for the phase.
     *
     * This copied and pasted from the Font Awesome page because it was easier to do that than try harder.
     * E.g.
     * * open the "times" page: https://fontawesome.com/icons/times?style=solid
     * * right click on the smallest icon (next to the unicode character) and view source.
     */
    // @ts-ignore
    private static iconPath(phase: NodePhase, complete: number) {
        switch (phase) {
            case 'Running':
                const radius = 250;
                const offset = (2 * Math.PI * 3) / 4;
                const theta0 = offset;
                // clip the line to min 5% max 95% so something always renders
                const theta1 = 2 * Math.PI * Math.max(0.05, Math.min(0.95, complete || 1)) + offset;
                const start = {x: 250 + radius * Math.cos(theta0), y: 250 + radius * Math.sin(theta0)};
                const end = {x: 250 + radius * Math.cos(theta1), y: 250 + radius * Math.sin(theta1)};
                const theta = theta1 - theta0;
                const largeArcFlag = theta > Math.PI ? 1 : 0;
                const sweepFlag = 1;
                return (
                    <path
                        stroke='currentColor'
                        strokeWidth={70}
                        fill='transparent'
                        d={`M${start.x},${start.y} A${radius},${radius} 0 ${largeArcFlag} ${sweepFlag} ${end.x},${end.y}`}
                    />
                );
        }
    }

    // @ts-ignore
    private static complete(node: NodeStatus) {
        if (!node || !node.estimatedDuration) {
            return null;
        }
        return (new Date().getTime() - new Date(node.startedAt).getTime()) / 1000 / node.estimatedDuration;
    }

    private graph: Graph;

    constructor(props: Readonly<WorkflowDagProps>) {
        super(props);
        this.state = {
            expandNodes: new Set()
        };
    }

    public render() {
        if (!this.props.nodes) {
            return <Loading />;
        }
        this.prepareGraph();

        return (
            <Ticker intervalMs={100}>
                {() => (
                    <GraphPanel
                        graph={this.graph}
                        filter={{types}}
                        onSelect={id => this.selectNode(id)}
                        nodeSize={48}
                        options={<WorkflowDagRenderOptionsPanel {...this.state} onChange={workflowDagRenderOptions => this.saveOptions(workflowDagRenderOptions)} />}
                    />
                )}
            </Ticker>
        );
    }

    private saveOptions(newChanges: WorkflowDagRenderOptions) {
        this.setState(newChanges);
    }

    private getNode(nodeId: string): NodeStatus {
        const node: NodeStatus = this.props.nodes[nodeId];
        if (!node) {
            return null;
        }
        return node;
    }

    private prepareGraph() {
        this.graph = new Graph();
        const edges = this.graph.edges;
        const nodes = this.graph.nodes;

        interface PrepareNode {
            nodeName: string;
            children: string[];
            parent: string;
        }

        const allNodes = this.props.nodes;
        const getChildren = (nodeId: string): string[] => {
            if (!allNodes[nodeId] || !allNodes[nodeId].children) {
                return [];
            }
            return allNodes[nodeId].children.filter(child => allNodes[child]);
        };
        const pushChildren = (nodeId: string, isExpanded: boolean, queue: PrepareNode[]): void => {
            const children: string[] = getChildren(nodeId);
            if (!children) {
                return;
            }

            if (children.length > 3 && !isExpanded) {
                // Node will be collapsed
                queue.push({
                    nodeName: children[0],
                    parent: nodeId,
                    children: getChildren(children[0])
                });
                const newChildren: string[] = children
                    .slice(1, children.length - 1)
                    .map(v => [v])
                    .reduce((a, b) => a.concat(b), []);
                queue.push({
                    nodeName: getCollapsedNodeName(nodeId, children.length - 2 + ' hidden nodes', allNodes[children[0]].type),
                    parent: nodeId,
                    children: newChildren
                });
                queue.push({
                    nodeName: children[children.length - 1],
                    parent: nodeId,
                    children: getChildren(children[children.length - 1])
                });
            } else {
                // Node will not be collapsed
                children.map(child =>
                    queue.push({
                        nodeName: child,
                        parent: nodeId,
                        children: getChildren(child)
                    })
                );
            }
        };

        const traverse = (root: PrepareNode): void => {
            const queue: PrepareNode[] = [root];
            const consideredChildren: Set<string> = new Set<string>();
            let previousCollapsed: string = '';

            while (queue.length > 0) {
                const item = queue.pop();

                const child = allNodes[item.nodeName];
                if (isCollapsedNode(item.nodeName)) {
                    if (item.nodeName !== previousCollapsed) {
                        nodes.set(item.nodeName, {
                            label: getMessage(item.nodeName),
                            type: child.type,
                            icon: icons.Collapsed
                        });
                        edges.set({v: item.parent, w: item.nodeName}, {});
                        previousCollapsed = item.nodeName;
                    }
                    continue;
                }

                const isExpanded: boolean = this.state.expandNodes.has('*') || this.state.expandNodes.has(item.nodeName);
                nodes.set(item.nodeName, nodeLabel(child));
                edges.set({v: item.parent, w: item.nodeName}, {});

                // If we have already considered the children of this node, don't consider them again
                if (consideredChildren.has(item.nodeName)) {
                    continue;
                }
                consideredChildren.add(item.nodeName);

                const node: NodeStatus = this.props.nodes[item.nodeName];
                if (!node || node.phase === NODE_PHASE.OMITTED) {
                    continue;
                }

                pushChildren(node.id, isExpanded, queue);
            }
        };

        const workflowRoot: PrepareNode = {
            nodeName: this.props.workflowName,
            parent: '',
            children: getChildren(this.props.workflowName)
        };

        // Traverse the workflow from the root node
        traverse(workflowRoot);

        const onExitHandlerNodeId = Object.values(allNodes).find(nodeId => nodeId.name === `${this.props.workflowName}.onExit`);
        if (onExitHandlerNodeId) {
            this.getOutboundNodes(this.props.workflowName).forEach(v => {
                const exitHandler = allNodes[onExitHandlerNodeId.id];
                nodes.set(onExitHandlerNodeId.id, nodeLabel(exitHandler));
                if (nodes.has(v)) {
                    edges.set({v, w: onExitHandlerNodeId.id}, {});
                }
            });
            const onExitRoot: PrepareNode = {
                nodeName: onExitHandlerNodeId.id,
                parent: '',
                children: getChildren(onExitHandlerNodeId.id)
            };
            // Traverse the onExit tree starting from the onExit node itself
            traverse(onExitRoot);
        }
    }

    private selectNode(nodeId: string) {
        if (isCollapsedNode(nodeId)) {
            this.expandNode(nodeId);
        } else {
            return this.props.nodeClicked && this.props.nodeClicked(nodeId);
        }
    }

    private expandNode(nodeId: string) {
        if (isCollapsedNode(getNodeParent(nodeId))) {
            this.expandNode(getNodeParent(nodeId));
        } else {
            this.setState({expandNodes: new Set(this.state.expandNodes).add(getNodeParent(nodeId))});
        }
    }

    private getOutboundNodes(nodeID: string): string[] {
        const node = this.getNode(nodeID);
        if (node.type === 'Pod' || node.type === 'Skipped') {
            return [node.id];
        }
        let outbound = Array<string>();
        for (const outboundNodeID of node.outboundNodes || []) {
            const outNode = this.getNode(outboundNodeID);
            if (outNode.type === 'Pod') {
                outbound.push(outboundNodeID);
            } else {
                outbound = outbound.concat(this.getOutboundNodes(outboundNodeID));
            }
        }
        return outbound;
    }
}
