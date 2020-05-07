import * as classNames from 'classnames';
import * as dagre from 'dagre';
import * as React from 'react';

import * as models from '../../../../models';
import {NodePhase, NodeStatus} from '../../../../models';
import {Loading} from '../../../shared/components/loading';
import {Utils} from '../../../shared/utils';
import {WorkflowDagRenderOptionsPanel} from './workflow-dag-render-options-panel';

export interface WorkflowDagRenderOptions {
    horizontal: boolean;
    scale: number;
    nodesToDisplay: string[];
}

export interface WorkflowDagProps {
    workflowName: string;
    nodes: {[nodeId: string]: NodeStatus};
    selectedNodeId?: string;
    nodeClicked?: (nodeId: string) => any;
}

require('./workflow-dag.scss');

export class WorkflowDag extends React.Component<WorkflowDagProps, WorkflowDagRenderOptions> {
    private get scale() {
        return this.state.scale;
    }

    private get nodeSize() {
        return 32 / this.scale;
    }

    /**
     * Return and SVG path for the phase.
     *
     * This copied and pasted from the Font Awesome page because it was easier to do that than try harder.
     * E.g.
     * * open the "times" page: https://fontawesome.com/icons/times?style=solid
     * * right click on the smallest icon (next to the unicode character) and view source.
     */
    private static iconPath(phase: string) {
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
            case 'Succeeded':
                return (
                    <path
                        fill='currentColor'
                        // tslint:disable-next-line
            d='M173.898 439.404l-166.4-166.4c-9.997-9.997-9.997-26.206 0-36.204l36.203-36.204c9.997-9.998 26.207-9.998 36.204 0L192 312.69 432.095 72.596c9.997-9.997 26.207-9.997 36.204 0l36.203 36.204c9.997 9.997 9.997 26.206 0 36.204l-294.4 294.401c-9.998 9.997-26.207 9.997-36.204-.001z'
                    />
                );
            case 'Running':
                return (
                    <path
                        fill='currentColor'
                        // tslint:disable-next-line
            d='M288 39.056v16.659c0 10.804 7.281 20.159 17.686 23.066C383.204 100.434 440 171.518 440 256c0 101.689-82.295 184-184 184-101.689 0-184-82.295-184-184 0-84.47 56.786-155.564 134.312-177.219C216.719 75.874 224 66.517 224 55.712V39.064c0-15.709-14.834-27.153-30.046-23.234C86.603 43.482 7.394 141.206 8.003 257.332c.72 137.052 111.477 246.956 248.531 246.667C393.255 503.711 504 392.788 504 256c0-115.633-79.14-212.779-186.211-240.236C302.678 11.889 288 23.456 288 39.056z'
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
                        {label.substr(0, label.length / 2 - 1)}
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

    constructor(props: Readonly<WorkflowDagProps>) {
        super(props);
        this.state = {
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
            ]
        };
    }

    public render() {
        if (!this.props.nodes) {
            return <Loading />;
        }
        const graph = new dagre.graphlib.Graph();
        // https://github.com/dagrejs/dagre/wiki
        graph.setGraph({
            edgesep: this.nodeSize,
            nodesep: this.nodeSize * 2,
            rankdir: this.state.horizontal ? 'LR' : 'TB',
            ranksep: this.nodeSize
        });
        graph.setDefaultEdgeLabel(() => ({}));
        const nodes = this.props.nodes;
        Object.values(nodes).forEach(node => {
            const label = Utils.shortNodeName(node);
            const nodeSize = this.filterNode(node) ? 1 : this.nodeSize;
            // one of the key improvements is passing less data to Dagre to layout
            graph.setNode(node.id, {label, id: node.id, width: nodeSize, height: nodeSize, phase: node.type === 'Suspend' && node.phase === 'Running' ? 'Suspended' : node.phase});
            (node.children || [])
                .map(childId => nodes[childId])
                .filter(child => child !== null)
                .forEach(child => graph.setEdge(node.id, child.id));
        });
        const onExitHandlerNodeId = Object.keys(nodes).find(id => nodes[id].name === `${this.props.workflowName}.onExit`);
        if (onExitHandlerNodeId) {
            this.getOutboundNodes(this.props.workflowName).forEach(nodeId => graph.setEdge(nodeId, onExitHandlerNodeId));
        }
        dagre.layout(graph);
        const size = this.getGraphSize(graph.nodes().map(id => graph.node(id)));
        return (
            <>
                <WorkflowDagRenderOptionsPanel {...this.state} onChange={workflowDagRenderOptions => this.setState(workflowDagRenderOptions)} />
                <div className='workflow-dag'>
                    <svg style={{width: size.width, height: size.height}}>
                        <defs>
                            <marker id='arrow' viewBox='0 0 10 10' refX={10} refY={5} markerWidth={this.nodeSize / 6} markerHeight={this.nodeSize / 6} orient='auto-start-reverse'>
                                <path d='M 0 0 L 10 5 L 0 10 z' className='arrow' />
                            </marker>
                            <filter id='shadow' x='0' y='0' width='200%' height='200%'>
                                <feOffset result='offOut' in='SourceGraphic' dx={0.5} dy={0.5} />
                                <feColorMatrix result='matrixOut' in='offOut' type='matrix' values='0.1 0 0 0 0 0 0.1 0 0 0 0 0 0.1 0 0 0 0 0 1 0' />
                                <feGaussianBlur result='blurOut' in='matrixOut' stdDeviation={0.5} />
                                <feBlend in='SourceGraphic' in2='blurOut' mode='normal' />
                            </filter>
                        </defs>
                        <g transform={`translate(${this.nodeSize},${this.nodeSize})`}>
                            {graph.edges().map(edge => {
                                const points = graph
                                    .edge(edge)
                                    .points.map((p, i) => (i === 0 ? `M ${p.x} ${p.y} ` : `L ${p.x} ${p.y}`))
                                    .join(' ');
                                return <path key={`line/${points}`} d={points} className='line' markerEnd={graph.node(edge.w).width > 1 && 'url(#arrow)'} />;
                            })}
                            {graph
                                .nodes()
                                .map(id => graph.node(id))
                                .map(node => (
                                    <g key={`node/${node.id}`} transform={`translate(${node.x},${node.y})`} onClick={() => this.selectNode(node.id)} className='node'>
                                        <circle
                                            r={node.width / 2}
                                            className={classNames(
                                                'workflow-dag__node',
                                                'workflow-dag__node-status',
                                                'workflow-dag__node-status--' + node.phase.toLocaleLowerCase(),
                                                {
                                                    active: node.id === this.props.selectedNodeId
                                                }
                                            )}
                                            filter='url(#shadow)'
                                        />
                                        {node.width > 1 && (
                                            <>
                                                {this.icon(node.phase)}
                                                <g transform={`translate(0,${node.height})`}>
                                                    <text className='label' fontSize={12 / this.scale}>
                                                        {WorkflowDag.formatLabel(node.label)}
                                                    </text>
                                                </g>
                                            </>
                                        )}
                                    </g>
                                ))}
                        </g>
                    </svg>
                </div>
            </>
        );
    }

    private selectNode(nodeId: string) {
        return this.props.nodeClicked && this.props.nodeClicked(nodeId);
    }

    private icon(phase: NodePhase) {
        return (
            <g>
                <g transform={`translate(-${this.nodeSize / 4},-${this.nodeSize / 4}), scale(${0.032 / this.scale})`} color='white'>
                    {WorkflowDag.iconPath(phase)}
                </g>
                {phase === 'Running' && (
                    <animateTransform attributeType='xml' attributeName='transform' type='rotate' from='0 0 0 ' to='360 0 0' dur='1s' additive='sum' repeatCount='indefinite' />
                )}
            </g>
        );
    }

    private getOutboundNodes(nodeID: string): string[] {
        const node = this.props.nodes[nodeID];
        if (node.type === 'Pod' || node.type === 'Skipped') {
            return [node.id];
        }
        let outbound = Array<string>();
        for (const outboundNodeID of node.outboundNodes || []) {
            const outNode = this.props.nodes[outboundNodeID];
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
            !(this.state.nodesToDisplay.includes('type:' + node.type) && this.state.nodesToDisplay.includes('phase:' + node.phase)) ||
            (node.type === 'Retry' && node.children.length === 1)
        );
    }

    private getGraphSize(nodes: dagre.Node[]): {width: number; height: number} {
        let width = 0;
        let height = 0;
        nodes.forEach(node => {
            width = Math.max(node.x + node.width, width);
            height = Math.max(node.y + node.height, height);
        });
        return {width: width + this.nodeSize * 2, height: height + this.nodeSize * 2};
    }
}
