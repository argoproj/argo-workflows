import * as React from 'react';

import {Artifact, ArtifactRepositoryRefStatus, NODE_PHASE, NodeStatus} from '../../../../models';
import {GraphPanel} from '../../../shared/components/graph/graph-panel';
import {Graph} from '../../../shared/components/graph/types';
import {Utils} from '../../../shared/utils';
import {genres} from './genres';
import {getCollapsedNodeName, getMessage, getNodeParent, isCollapsedNode} from './graph/collapsible-node';
import {icons} from './icons';
import {WorkflowDagRenderOptionsPanel} from './workflow-dag-render-options-panel';

export interface WorkflowDagRenderOptions {
    expandNodes: Set<string>;
}

interface WorkflowDagProps {
    workflowName: string;
    showArtifacts: boolean;
    artifactRepositoryRef?: ArtifactRepositoryRefStatus;
    nodes: {[nodeId: string]: NodeStatus};
    selectedNodeId?: string;
    nodeSize?: number;
    hideOptions?: boolean;
    nodeClicked?: (nodeId: string) => any;
}

function progress(n: NodeStatus) {
    if (!n || !n.estimatedDuration) {
        return null;
    }
    return (new Date().getTime() - new Date(n.startedAt).getTime()) / 1000 / n.estimatedDuration;
}

function getNodeLabelTemplateName(n: NodeStatus): string {
    return n.templateName || (n.templateRef && n.templateRef.template + '/' + n.templateRef.name) || 'no template';
}

function nodeLabel(n: NodeStatus) {
    const phase = n.type === 'Suspend' && n.phase === 'Running' ? 'Suspended' : n.phase;
    return {
        label: Utils.shortNodeName(n),
        genre: n.type,
        icon: icons[phase] || icons.Pending,
        progress: phase === 'Running' && progress(n),
        classNames: phase,
        tags: new Set([getNodeLabelTemplateName(n)])
    };
}

const classNames = (() => {
    const v: {[label: string]: boolean} = {
        Artifact: true,
        Suspended: true,
        Collapsed: true
    };
    Object.entries(NODE_PHASE).forEach(([, label]) => (v[label] = true));
    return v;
})();

export class WorkflowDag extends React.Component<WorkflowDagProps, WorkflowDagRenderOptions> {
    private graph: Graph;

    constructor(props: Readonly<WorkflowDagProps>) {
        super(props);
        this.state = {
            expandNodes: new Set()
        };
    }

    public render() {
        this.prepareGraph();

        const tags: {[key: string]: boolean} = {};
        Object.values(this.props.nodes || {}).forEach(n => (tags[getNodeLabelTemplateName(n)] = true));

        return (
            <GraphPanel
                storageScope='workflow-dag'
                graph={this.graph}
                nodeGenresTitle={'Node Type'}
                nodeGenres={genres}
                nodeClassNamesTitle={'Node Phase'}
                nodeClassNames={classNames}
                nodeTagsTitle={'Template'}
                nodeTags={tags}
                nodeSize={this.props.nodeSize || 32}
                defaultIconShape='circle'
                hideNodeTypes={true}
                hideOptions={this.props.hideOptions}
                selectedNode={this.props.selectedNodeId}
                onNodeSelect={id => this.selectNode(id)}
                options={<WorkflowDagRenderOptionsPanel {...this.state} onChange={workflowDagRenderOptions => this.saveOptions(workflowDagRenderOptions)} />}
            />
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

        if (!this.props.nodes) {
            return;
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

                if (isCollapsedNode(item.nodeName)) {
                    if (item.nodeName !== previousCollapsed) {
                        nodes.set(item.nodeName, {
                            label: getMessage(item.nodeName),
                            genre: 'Collapsed',
                            icon: icons.Collapsed,
                            classNames: 'Collapsed'
                        });
                        edges.set({v: item.parent, w: item.nodeName}, {});
                        previousCollapsed = item.nodeName;
                    }
                    continue;
                }
                const child = allNodes[item.nodeName];
                if (!child) {
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

        if (this.props.showArtifacts) {
            Object.values(this.props.nodes)
                .filter(node => nodes.has(node.id))
                .forEach(node => {
                    (node.inputs?.artifacts || [])
                        .map(a => this.generifyArtifact(a))
                        .forEach(a => {
                            nodes.set(a.id, a.label);
                            edges.set(
                                {v: a.id, w: node.id},
                                {
                                    label: a.name
                                }
                            );
                        });
                    (node.outputs?.artifacts || [])
                        .filter(a => !a.name.endsWith('-logs'))
                        .map(a => this.generifyArtifact(a))
                        .forEach(a => {
                            nodes.set(a.id, a.label);
                            edges.set({v: node.id, w: a.id}, {label: a.name});
                        });
                });
        }
    }

    private generifyArtifact(a: Artifact) {
        let id = 'unknown';
        let label = 'unknown';
        if (a.gcs) {
            label = a.gcs.key;
            id = 'artifact:gcs:' + (a.gcs.endpoint || this.artifactRepository.gcs?.endpoint) + ':' + (a.gcs.bucket || this.artifactRepository.gcs?.bucket) + ':' + label;
        }
        if (a.git) {
            const revision = a.git.revision || 'HEAD';
            label = a.git.repo + '#' + revision;
            id = 'artifact:git:' + a.git.repo + ':' + revision;
        }
        if (a.http) {
            label = a.http.url;
            id = 'artifact:http::' + a.http.url;
        }
        if (a.s3) {
            label = a.s3.key;
            id = 'artifact:s3:' + (a.s3.endpoint || this.artifactRepository.s3.endpoint) + ':' + (a.s3?.bucket || this.artifactRepository.s3?.bucket) + ':' + label;
        }
        if (a.oss) {
            label = a.oss.key;
            id = 'artifact:oss:' + (a.oss.endpoint || this.artifactRepository.oss?.endpoint) + ':' + (a.oss.bucket || this.artifactRepository.oss?.bucket) + ':' + label;
        }
        if (a.raw) {
            label = 'raw';
            id = 'artifact:raw:' + a.raw.data;
        }
        return {
            id,
            name: a.name,
            label: {
                genre: 'Artifact',
                label,
                icon: icons.Artifact,
                classNames: 'Artifact'
            }
        };
    }

    private get artifactRepository() {
        return this.props.artifactRepositoryRef?.artifactRepository || {};
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
