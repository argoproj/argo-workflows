import React, {useMemo, useState} from 'react';

import {ArtifactRepositoryRefStatus, NODE_PHASE, NodeStatus} from '../../../../models';
import {nodeArtifacts} from '../../../shared/artifacts';
import {GraphPanel} from '../../../shared/components/graph/graph-panel';
import {Graph} from '../../../shared/components/graph/types';
import {shortNodeName} from '../../utils';
import {genres} from './genres';
import {getCollapsedNodeName, getMessage, getNodeParent, isCollapsedNode} from './graph/collapsible-node';
import {icons} from './icons';
import {WorkflowDagRenderOptionsPanel} from './workflow-dag-render-options-panel';

export interface WorkflowDagRenderOptions {
    expandNodes: Set<string>;
    showArtifacts: boolean;
}

interface WorkflowDagProps {
    workflowName: string;
    artifactRepositoryRef?: ArtifactRepositoryRefStatus;
    nodes: Record<string, NodeStatus>;
    selectedNodeId?: string;
    nodeSize?: number;
    hideOptions?: boolean;
    nodeClicked?: (nodeId: string) => void;
}

interface PrepareNode {
    nodeName: string;
    children: string[];
    parent: string;
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
        label: shortNodeName(n),
        genre: n.type,
        icon: icons[phase] || icons.Pending,
        progress: phase === 'Running' && progress(n),
        classNames: phase,
        tags: new Set([getNodeLabelTemplateName(n)])
    };
}

const classNames = {
    Artifact: true,
    Suspended: true,
    Collapsed: true,
    ...Object.entries(NODE_PHASE).reduce((acc, [, value]) => ({...acc, [value]: true}), {})
};

export function WorkflowDag(props: WorkflowDagProps) {
    const artifactRepository = props.artifactRepositoryRef?.artifactRepository || {};
    const tags: Record<string, boolean> = Object.values(props.nodes || {}).reduce((acc, node) => ({...acc, [getNodeLabelTemplateName(node)]: true}), {});

    const [expandNodes, setExpandNodes] = useState(new Set<string>());
    const [showArtifacts, setShowArtifacts] = useState(localStorage.getItem('showArtifacts') !== 'false');

    const graph = useMemo(() => {
        const newGraph = new Graph();
        const {edges, nodes} = newGraph;

        if (!props.nodes) {
            return newGraph;
        }

        const allNodes = props.nodes;

        // Traverse the workflow from the root node
        traverse({
            nodeName: props.workflowName,
            parent: '',
            children: getChildren(props.workflowName)
        });

        const onExitHandlerNode = Object.values(allNodes).find(node => node.name === `${props.workflowName}.onExit`);
        if (onExitHandlerNode) {
            getOutboundNodes(props.workflowName).forEach(v => {
                const exitHandler = allNodes[onExitHandlerNode.id];
                nodes.set(onExitHandlerNode.id, nodeLabel(exitHandler));
                if (nodes.has(v)) {
                    edges.set({v, w: onExitHandlerNode.id}, {});
                }
            });
            // Traverse the onExit tree starting from the onExit node itself
            traverse({
                nodeName: onExitHandlerNode.id,
                parent: '',
                children: getChildren(onExitHandlerNode.id)
            });
        }

        if (showArtifacts) {
            Object.values(props.nodes || {})
                .filter(node => nodes.has(node.id))
                .forEach(node => {
                    nodeArtifacts(node, artifactRepository)
                        .filter(({name}) => !name.endsWith('-logs'))
                        // only show files or directories
                        .filter(({filename, key}) => filename.includes('.') || key.endsWith('/'))
                        .forEach(a => {
                            nodes.set(a.urn, {
                                genre: 'Artifact',
                                label: a.filename,
                                icon: icons.Artifact,
                                classNames: 'Artifact'
                            });
                            const input = a.artifactNameDiscriminator === 'input';
                            edges.set(
                                {v: input ? a.urn : node.id, w: input ? node.id : a.urn},
                                {
                                    label: a.name,
                                    classNames: 'related'
                                }
                            );
                        });
                });
        }

        function getChildren(nodeId: string): string[] {
            if (!allNodes[nodeId] || !allNodes[nodeId].children) {
                return [];
            }
            return allNodes[nodeId].children.filter(child => allNodes[child]);
        }

        function pushChildren(nodeId: string, isExpanded: boolean, queue: PrepareNode[]): void {
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
        }

        function traverse(root: PrepareNode): void {
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
                const isExpanded = expandNodes.has('*') || expandNodes.has(item.nodeName);

                nodes.set(item.nodeName, nodeLabel(child));
                edges.set({v: item.parent, w: item.nodeName}, {});

                // If we have already considered the children of this node, don't consider them again
                if (consideredChildren.has(item.nodeName)) {
                    continue;
                }
                consideredChildren.add(item.nodeName);

                const node = getNode(item.nodeName);
                if (!node) {
                    continue;
                }

                pushChildren(node.id, isExpanded, queue);
            }
        }

        return newGraph;
    }, [props.nodes, expandNodes, showArtifacts]);

    function getNode(nodeId: string) {
        return props.nodes[nodeId];
    }

    function expandNode(nodeId: string) {
        if (isCollapsedNode(getNodeParent(nodeId))) {
            expandNode(getNodeParent(nodeId));
        } else {
            setExpandNodes(new Set(expandNodes).add(getNodeParent(nodeId)));
        }
    }

    function getOutboundNodes(nodeID: string): string[] {
        const node = getNode(nodeID);
        if (!node) {
            return [];
        }
        if (node.type === 'Pod' || node.type === 'Skipped') {
            return [node.id];
        }
        let outbound: string[];
        for (const outboundNodeID of node.outboundNodes || []) {
            const outNode = getNode(outboundNodeID);
            if (outNode?.type === 'Pod') {
                outbound.push(outboundNodeID);
            } else {
                outbound = outbound.concat(getOutboundNodes(outboundNodeID));
            }
        }
        return outbound;
    }

    return (
        <GraphPanel
            storageScope='workflow-dag'
            graph={graph}
            nodeGenresTitle={'Node Type'}
            nodeGenres={genres}
            nodeClassNamesTitle={'Node Phase'}
            nodeClassNames={classNames}
            nodeTagsTitle={'Template'}
            nodeTags={tags}
            nodeSize={props.nodeSize || 32}
            defaultIconShape='circle'
            hideNodeTypes={true}
            hideOptions={props.hideOptions}
            selectedNode={props.selectedNodeId}
            onNodeSelect={nodeId => {
                if (isCollapsedNode(nodeId)) {
                    expandNode(nodeId);
                } else {
                    return props?.nodeClicked(nodeId);
                }
            }}
            options={
                <WorkflowDagRenderOptionsPanel
                    expandNodes={expandNodes}
                    showArtifacts={showArtifacts}
                    onChange={newOptions => {
                        localStorage.setItem('showArtifacts', newOptions.showArtifacts ? 'true' : 'false');
                        setExpandNodes(newOptions.expandNodes);
                        setShowArtifacts(newOptions.showArtifacts);
                    }}
                />
            }
        />
    );
}
