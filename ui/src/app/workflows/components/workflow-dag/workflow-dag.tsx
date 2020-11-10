import {Ticker} from 'argo-ui';
import * as classNames from 'classnames';
import * as dagre from 'dagre';
import * as React from 'react';

import {NODE_PHASE, NodePhase, NodeStatus} from '../../../../models';
import {Loading} from '../../../shared/components/loading';
import {Utils} from '../../../shared/utils';
import {CoffmanGrahamSorter} from './graph/coffman-graham-sorter';
import {getCollapsedNodeName, getMessage, getNodeParent, getType, isCollapsedNode} from './graph/collapsible-node';
import {Edge, Graph} from './graph/graph';
import {WorkflowDagRenderOptionsPanel} from './workflow-dag-render-options-panel';

export interface WorkflowDagRenderOptions {
    horizontal: boolean;
    scale: number;
    nodesToDisplay: string[];
    expandNodes: Set<string>;
    fastRenderer: boolean;
}

export interface WorkflowDagProps {
    workflowName: string;
    nodes: {[nodeId: string]: NodeStatus};
    selectedNodeId?: string;
    nodeClicked?: (nodeId: string) => any;
}

require('./workflow-dag.scss');

type DagPhase = NodePhase | 'Suspended' | 'Collapsed-Horizontal' | 'Collapsed-Vertical';

const LOCAL_STORAGE_KEY = 'DagOptions';

export class WorkflowDag extends React.Component<WorkflowDagProps, WorkflowDagRenderOptions> {
    private get scale() {
        return this.state.scale;
    }

    private get nodeSize() {
        return 32 / this.scale;
    }

    private get gap() {
        return 1.25 * this.nodeSize;
    }

    /**
     * Return and SVG path for the phase.
     *
     * This copied and pasted from the Font Awesome page because it was easier to do that than try harder.
     * E.g.
     * * open the "times" page: https://fontawesome.com/icons/times?style=solid
     * * right click on the smallest icon (next to the unicode character) and view source.
     */
    private static iconPath(phase: DagPhase, complete: number) {
        switch (phase) {
            case 'Pending':
                return (
                    <path
                        fill='currentColor'
                        // tslint:disable-next-line
                        d='M256,8C119,8,8,119,8,256S119,504,256,504,504,393,504,256,393,8,256,8Zm92.49,313h0l-20,25a16,16,0,0,1-22.49,2.5h0l-67-49.72a40,40,0,0,1-15-31.23V112a16,16,0,0,1,16-16h32a16,16,0,0,1,16,16V256l58,42.5A16,16,0,0,1,348.49,321Z'
                    />
                );
            case 'Failed':
            case 'Error':
                return (
                    <g transform='translate(60,0)'>
                        <path
                            fill='currentColor'
                            // tslint:disable-next-line
                            d='M242.72 256l100.07-100.07c12.28-12.28 12.28-32.19 0-44.48l-22.24-22.24c-12.28-12.28-32.19-12.28-44.48 0L176 189.28 75.93 89.21c-12.28-12.28-32.19-12.28-44.48 0L9.21 111.45c-12.28 12.28-12.28 32.19 0 44.48L109.28 256 9.21 356.07c-12.28 12.28-12.28 32.19 0 44.48l22.24 22.24c12.28 12.28 32.2 12.28 44.48 0L176 322.72l100.07 100.07c12.28 12.28 32.2 12.28 44.48 0l22.24-22.24c12.28-12.28 12.28-32.19 0-44.48L242.72 256z'
                        />
                    </g>
                );
            case 'Skipped':
                return (
                    <path
                        fill='currentColor'
                        // tslint:disable-next-line
                        d='M500.5 231.4l-192-160C287.9 54.3 256 68.6 256 96v320c0 27.4 31.9 41.8 52.5 24.6l192-160c15.3-12.8 15.3-36.4 0-49.2zm-256 0l-192-160C31.9 54.3 0 68.6 0 96v320c0 27.4 31.9 41.8 52.5 24.6l192-160c15.3-12.8 15.3-36.4 0-49.2z'
                    />
                );
            case 'Omitted':
                return (
                    <path
                        fill='currentColor'
                        // tslint:disable-next-line
                        d='M500.5 231.4l-192-160C287.9 54.3 256 68.6 256 96v320c0 27.4 31.9 41.8 52.5 24.6l192-160c15.3-12.8 15.3-36.4 0-49.2zm-256 0l-192-160C31.9 54.3 0 68.6 0 96v320c0 27.4 31.9 41.8 52.5 24.6l192-160c15.3-12.8 15.3-36.4 0-49.2z'
                    />
                );
            case 'Succeeded':
                return (
                    <path
                        fill='currentColor'
                        // tslint:disable-next-line
                        d='M173.898 439.404l-166.4-166.4c-9.997-9.997-9.997-26.206 0-36.204l36.203-36.204c9.997-9.998 26.207-9.998 36.204 0L192 312.69 432.095 72.596c9.997-9.997 26.207-9.997 36.204 0l36.203 36.204c9.997 9.997 9.997 26.206 0 36.204l-294.4 294.401c-9.998 9.997-26.207 9.997-36.204-.001z'
                    />
                );
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
            case 'Suspended':
                return (
                    <path
                        fill='currentColor'
                        // tslint:disable-next-line
                        d='M144 479H48c-26.5 0-48-21.5-48-48V79c0-26.5 21.5-48 48-48h96c26.5 0 48 21.5 48 48v352c0 26.5-21.5 48-48 48zm304-48V79c0-26.5-21.5-48-48-48h-96c-26.5 0-48 21.5-48 48v352c0 26.5 21.5 48 48 48h96c26.5 0 48-21.5 48-48z'
                    />
                );
            case 'Collapsed-Horizontal':
                return (
                    <path
                        fill='currentColor'
                        // tslint:disable-next-line
                        d='M328 256c0 39.8-32.2 72-72 72s-72-32.2-72-72 32.2-72 72-72 72 32.2 72 72zm104-72c-39.8 0-72 32.2-72 72s32.2 72 72 72 72-32.2 72-72-32.2-72-72-72zm-352 0c-39.8 0-72 32.2-72 72s32.2 72 72 72 72-32.2 72-72-32.2-72-72-72z'
                    />
                );
            case 'Collapsed-Vertical':
                return (
                    <path
                        fill='currentColor'
                        transform='translate(150,0)'
                        // tslint:disable-next-line
                        d='M96 184c39.8 0 72 32.2 72 72s-32.2 72-72 72-72-32.2-72-72 32.2-72 72-72zM24 80c0 39.8 32.2 72 72 72s72-32.2 72-72S135.8 8 96 8 24 40.2 24 80zm0 352c0 39.8 32.2 72 72 72s72-32.2 72-72-32.2-72-72-72-72 32.2-72 72z'
                    />
                );
        }
    }

    private static formatLabel(label: string) {
        const maxPerLine = 14;
        if (label.length <= maxPerLine) {
            return <tspan>{label}</tspan>;
        }
        if (label.length <= maxPerLine * 2) {
            return (
                <>
                    <tspan x={0} dy='-0.2em'>
                        {label.substr(0, label.length / 2)}
                    </tspan>
                    <tspan x={0} dy='1.2em'>
                        {label.substr(label.length / 2)}
                    </tspan>
                </>
            );
        }
        return (
            <>
                <tspan x={0} dy='-0.2em'>
                    {label.substr(0, maxPerLine - 2)}..
                </tspan>
                <tspan x={0} dy='1.2em'>
                    {label.substr(label.length + 1 - maxPerLine)}
                </tspan>
            </>
        );
    }

    private static complete(node: NodeStatus) {
        if (!node || !node.estimatedDuration) {
            return null;
        }
        return (new Date().getTime() - new Date(node.startedAt).getTime()) / 1000 / node.estimatedDuration;
    }

    private hash: {scale: number; nodeCount: number; nodesToDisplay: string[]};
    private graph: {
        width: number;
        height: number;
        edges: {v: string; w: string; points: {x: number; y: number}[]}[];
        nodes: Map<string, {x: number; y: number}>;
    };

    constructor(props: Readonly<WorkflowDagProps>) {
        super(props);
        this.state = {
            ...this.getOptions(),
            expandNodes: new Set()
        };
    }

    public render() {
        if (!this.props.nodes) {
            return <Loading />;
        }
        const {nodes, edges} = this.prepareGraph();
        this.layoutGraph(nodes, edges);

        return (
            <>
                <WorkflowDagRenderOptionsPanel {...this.state} onChange={workflowDagRenderOptions => this.saveOptions(workflowDagRenderOptions)} />
                <Ticker intervalMs={100}>
                    {() => (
                        <div className='workflow-dag'>
                            <svg
                                style={{
                                    width: this.graph.width + this.gap * 2,
                                    height: this.graph.height + this.gap * 2,
                                    margin: this.nodeSize
                                }}>
                                <defs>
                                    <marker
                                        id='arrow'
                                        viewBox='0 0 10 10'
                                        refX={10}
                                        refY={5}
                                        markerWidth={this.nodeSize / 6}
                                        markerHeight={this.nodeSize / 6}
                                        orient='auto-start-reverse'>
                                        <path d='M 0 0 L 10 5 L 0 10 z' className='arrow' />
                                    </marker>
                                    <filter id='shadow' x='0' y='0' width='200%' height='200%'>
                                        <feOffset result='offOut' in='SourceGraphic' dx={0.5} dy={0.5} />
                                        <feColorMatrix result='matrixOut' in='offOut' type='matrix' values='0.1 0 0 0 0 0 0.1 0 0 0 0 0 0.1 0 0 0 0 0 1 0' />
                                        <feGaussianBlur result='blurOut' in='matrixOut' stdDeviation={0.5} />
                                        <feBlend in='SourceGraphic' in2='blurOut' mode='normal' />
                                    </filter>
                                </defs>
                                <g transform={`translate(${this.gap},${this.gap})`}>
                                    {this.graph.edges.map(edge => {
                                        const points = edge.points.map((p, i) => (i === 0 ? `M ${p.x} ${p.y} ` : `L ${p.x} ${p.y}`)).join(' ');
                                        return <path key={`line/${edge.v}-${edge.w}`} d={points} className='line' markerEnd={this.hiddenNode(edge.w) ? '' : 'url(#arrow)'} />;
                                    })}
                                    {Array.from(this.graph.nodes).map(([nodeId, v]) => {
                                        let phase: DagPhase;
                                        let label: string;
                                        let hidden: boolean;
                                        const node = this.getNode(nodeId);
                                        if (isCollapsedNode(nodeId)) {
                                            phase = this.state.horizontal ? 'Collapsed-Vertical' : 'Collapsed-Horizontal';
                                            label = getMessage(nodeId);
                                            hidden = this.hiddenNode(nodeId);
                                        } else {
                                            phase = node.type === 'Suspend' && node.phase === 'Running' ? 'Suspended' : node.phase;
                                            label = Utils.shortNodeName(node);
                                            hidden = this.hiddenNode(nodeId);
                                        }
                                        return (
                                            <g key={`node/${nodeId}`} transform={`translate(${v.x},${v.y})`} onClick={() => this.selectNode(nodeId)} className='node'>
                                                <title>{label}</title>
                                                <circle
                                                    r={this.nodeSize / (hidden ? 16 : 2)}
                                                    className={classNames('workflow-dag__node', 'workflow-dag__node-status', 'workflow-dag__node-status--' + phase.toLowerCase(), {
                                                        active: nodeId === this.props.selectedNodeId,
                                                        hidden
                                                    })}
                                                />
                                                {!hidden && (
                                                    <>
                                                        {this.icon(phase, WorkflowDag.complete(node))}
                                                        <g transform={`translate(0,${this.nodeSize})`}>
                                                            <text className='label' fontSize={12 / this.scale}>
                                                                {WorkflowDag.formatLabel(label)}
                                                            </text>
                                                        </g>
                                                    </>
                                                )}
                                            </g>
                                        );
                                    })}
                                </g>
                            </svg>
                        </div>
                    )}
                </Ticker>
            </>
        );
    }

    private saveOptions(newChanges: WorkflowDagRenderOptions) {
        localStorage.setItem(LOCAL_STORAGE_KEY, JSON.stringify(newChanges));
        this.setState(newChanges);
    }

    private getNode(nodeId: string): NodeStatus {
        const node: NodeStatus = this.props.nodes[nodeId];
        if (!node) {
            return null;
        }
        return node;
    }

    private getOptions(): WorkflowDagRenderOptions {
        if (localStorage.getItem(LOCAL_STORAGE_KEY) !== null) {
            return JSON.parse(localStorage.getItem(LOCAL_STORAGE_KEY)) as WorkflowDagRenderOptions;
        }
        return {
            horizontal: false,
            scale: 1,
            nodesToDisplay: [
                'phase:Pending',
                'phase:Running',
                'phase:Succeeded',
                'phase:Skipped',
                'phase:Failed',
                'phase:Error',
                'type:Pod',
                'type:Steps',
                'type:DAG',
                'type:Retry',
                'type:Skipped',
                'type:Suspend'
            ],
            fastRenderer: false
        } as WorkflowDagRenderOptions;
    }

    private prepareGraph() {
        const edges: Edge[] = [];
        const nodes: string[] = [];

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

                if (isCollapsedNode(item.nodeName)) {
                    if (item.nodeName !== previousCollapsed) {
                        nodes.push(item.nodeName);
                        edges.push({v: item.parent, w: item.nodeName});
                        previousCollapsed = item.nodeName;
                    }
                    continue;
                }

                const isExpanded: boolean = this.state.expandNodes.has('*') || this.state.expandNodes.has(item.nodeName);
                nodes.push(item.nodeName);
                edges.push({v: item.parent, w: item.nodeName});

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
                nodes.push(onExitHandlerNodeId.id);
                if (nodes.includes(v)) {
                    edges.push({v, w: onExitHandlerNodeId.id});
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
        return {nodes, edges};
    }

    private layoutGraphPretty(nodes: string[], edges: Edge[]) {
        const graph = new dagre.graphlib.Graph();

        graph.setGraph({
            edgesep: 2.5 * this.gap,
            nodesep: this.gap,
            rankdir: this.state.horizontal ? 'LR' : 'TB',
            ranksep: this.gap
        });

        graph.setDefaultEdgeLabel(() => ({}));

        nodes.forEach(node => {
            graph.setNode(node, {label: node, width: this.nodeSize, height: this.nodeSize});
        });
        edges.forEach(edge => {
            if (edge.v && edge.w && graph.node(edge.v) && graph.node(edge.w)) {
                graph.setEdge(edge.v, edge.w);
            }
        });
        dagre.layout(graph);
        const size = this.getGraphSize(graph.nodes().map((id: string) => graph.node(id)));
        this.graph = {
            width: size.width,
            height: size.height,
            nodes: new Map<string, {x: number; y: number}>(),
            edges: []
        };
        graph
            .nodes()
            .map(id => graph.node(id))
            .forEach(node => {
                this.graph.nodes.set(node.label, {x: node.x, y: node.y});
            });
        graph.edges().forEach((edge: Edge) => {
            this.graph.edges.push(this.generateEdge(edge));
        });
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

    private layoutGraphFast(nodes: string[], edges: Edge[]) {
        const g = new Graph();
        g.nodes = nodes;
        g.edges = new Set(edges);
        const layers = new CoffmanGrahamSorter(g).sort();

        this.graph = {
            width: 0,
            height: 0,
            nodes: new Map<string, {x: number; y: number}>(),
            edges: []
        };
        // we have a lot of logic here about laying it out with suitable gaps - but what if we
        // would just translate it somehow?
        if (this.state.horizontal) {
            this.graph.width = layers.length * this.gap * 2;
        } else {
            this.graph.height = layers.length * this.gap * 2;
        }
        layers.forEach(level => {
            if (this.state.horizontal) {
                this.graph.height = Math.max(this.graph.height, level.length * this.gap * 2);
            } else {
                this.graph.width = Math.max(this.graph.width, level.length * this.gap * 2);
            }
        });
        layers.forEach((level, i) => {
            level.forEach((node, j) => {
                const l = this.state.horizontal ? 0 : this.graph.width / 2 - level.length * this.gap;
                const t = !this.state.horizontal ? 0 : this.graph.height / 2 - level.length * this.gap;
                this.graph.nodes.set(node, {
                    x: (this.state.horizontal ? i : j) * this.gap * 2 + l,
                    y: (this.state.horizontal ? j : i) * this.gap * 2 + t
                });
            });
        });
        this.graph.edges = edges.filter(e => this.graph.nodes.has(e.v) && this.graph.nodes.has(e.w)).map(e => this.generateEdge(e));
    }

    private generateEdge(edge: Edge) {
        // `h` and `v` move the arrow heads to next to the node, otherwise they would be behind it
        const h = this.state.horizontal ? this.nodeSize / 2 : 0;
        const v = !this.state.horizontal ? this.nodeSize / 2 : 0;
        return {
            v: edge.v,
            w: edge.w,
            points: [
                {
                    // for hidden nodes, we want to size them zero
                    x: this.graph.nodes.get(edge.v).x + (this.hiddenNode(edge.v) ? 0 : h),
                    y: this.graph.nodes.get(edge.v).y + (this.hiddenNode(edge.v) ? 0 : v)
                },
                {
                    x: this.graph.nodes.get(edge.w).x - (this.hiddenNode(edge.w) ? 0 : h),
                    y: this.graph.nodes.get(edge.w).y - (this.hiddenNode(edge.w) ? 0 : v)
                }
            ]
        };
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

    private icon(phase: DagPhase, complete: number) {
        return (
            <g>
                <g transform={`translate(-${this.nodeSize / 4},-${this.nodeSize / 4}), scale(${0.032 / this.scale})`} color='white'>
                    {WorkflowDag.iconPath(phase, complete)}
                </g>
                {phase === 'Running' && (!complete || complete >= 1) && (
                    <animateTransform attributeType='xml' attributeName='transform' type='rotate' from='0 0 0 ' to='360 0 0' dur='2s' additive='sum' repeatCount='indefinite' />
                )}
            </g>
        );
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

    private hiddenNode(id: string): boolean {
        if (isCollapsedNode(id)) {
            return !this.state.nodesToDisplay.includes('type:' + getType(id));
        }

        const node = this.getNode(id);
        // Filter the node if it is a virtual node or a Retry node with one child
        return (
            !(this.state.nodesToDisplay.includes('type:' + node.type) && this.state.nodesToDisplay.includes('phase:' + node.phase)) ||
            (node.type === 'Retry' && node.children && node.children.length === 1)
        );
    }

    private layoutGraph(nodes: string[], edges: Edge[]) {
        const hash = {scale: this.scale, nodeCount: nodes.length, nodesToDisplay: this.state.nodesToDisplay};
        // this hash check prevents having to do the expensive layout operation, if the graph does not re-laying out (e.g. phase change only)
        if (this.hash === hash) {
            return;
        }
        this.hash = hash;
        if (this.state.fastRenderer) {
            this.layoutGraphFast(nodes, edges);
        } else {
            this.layoutGraphPretty(nodes, edges);
        }
    }
}
