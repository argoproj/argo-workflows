import * as dagre from 'dagre';
import * as React from 'react';

require('./graph-panel.scss');

export class Node {
    public id: string;
    public type: string;
    public label: string;
    public icon: string;
    public phase?: string;
    public touched?: boolean;
}

export interface Graph {
    nodes?: Node[];
    edges?: Edge[];
}

export interface Edge {
    x: string;
    y: string;
    label?: string;
}

interface Props {
    graph: Graph;
    onSelect: (id: string) => void;
}

const icons: {[key: string]: string} = {
    'bell': '\uf0f3',
    'calendar': '\uf133',
    'cloud': '\uf0c2',
    'circle': '\uf111',
    'clock': '\uf017',
    'code': '\uf121',
    'comment': '\uf075',
    'code-branch': '\uf126',
    'credit-card': '\uf09d',
    'database': '\uf1c0',
    'envelope': '\uf0e0',
    'file': '\uf15b',
    'file-code': '\uf1c9',
    'filter': '\uf0b0',
    'hdd': '\uf0a0',
    'link': '\uf0c1',
    'microchip': '\uf2db',
    'puzzle-piece': '\uf12e',
    'save': '\uf0c7',
    'sitemap': '\uf0e8',
    'stream': '\uf550',
    'th': '\uf00a'
};

export class GraphPanel extends React.Component<Props> {
    constructor(props: Readonly<Props>) {
        super(props);
    }

    public render() {
        const nodeSize = 64;
        const ranksep = nodeSize * 2;
        const nodesep = ranksep;
        const g = new dagre.graphlib.Graph();
        g.setGraph({rankdir: 'LR', ranksep, nodesep});
        g.setDefaultEdgeLabel(() => ({}));
        (this.props.graph.nodes || []).forEach(n =>
            g.setNode(n.id, {
                label: n.label,
                type: n.type,
                icon: n.icon,
                phase: n.phase,
                touched: n.touched,
                width: nodeSize,
                height: nodeSize
            })
        );
        (this.props.graph.edges || []).forEach(e => g.setEdge(e.x, e.y, {label: e.label}));

        dagre.layout(g);

        const nodes = g
            .nodes()
            .map(id => ({...g.node(id), ...{id}}))
            .filter(n => n.x);
        const width = nodes.map(n => n.x + n.width + 2 * nodeSize).reduce((l, r) => Math.max(l, r), 0);
        const height = nodes.map(n => n.y + n.height + 2 * nodeSize).reduce((l, r) => Math.max(l, r), 0);

        return (
            <svg key='graph' className='graph' style={{width, height, margin: nodeSize}}>
                <defs>
                    <marker id='arrow' viewBox='0 0 10 10' refX={10} refY={5} markerWidth={nodeSize / 8} markerHeight={nodeSize / 8} orient='auto-start-reverse'>
                        <path d='M 0 0 L 10 5 L 0 10 z' className='arrow' />
                    </marker>
                </defs>
                <g transform={`translate(${nodeSize},${nodeSize})`}>
                    {g
                        .edges()
                        .map(e => g.edge(e))
                        .map((e, i) => (
                            <>
                                <path
                                    key={`edge/${i}`}
                                    d={e.points.map((p, j) => (j === 0 ? `M ${p.x} ${p.y} ` : `L ${p.x} ${p.y}`)).join(' ')}
                                    className='line'
                                    markerEnd='url(#arrow)'
                                />
                                <g transform={`translate(${e.points[1].x},${e.points[1].y})`}>
                                    <text className='label'>{e.label}</text>
                                </g>
                            </>
                        ))}
                    {nodes
                        .filter(n => n.x)
                        .map((n: any) => (
                            <g key={`node/${n.id}`} transform={`translate(${n.x},${n.y})`} className='node'>
                                <title>{n.id}</title>
                                <g className={`icon ${n.phase} ${n.touched && 'touched'}`} onClick={() => this.props.onSelect(n.id)}>
                                    <rect x={-nodeSize / 2} y={-nodeSize / 2} width={nodeSize} height={nodeSize} rx={nodeSize / 8} ry={nodeSize / 8} className='bg' />
                                    <text>
                                        <tspan x={0} y={4} className='icon'>
                                            {icons[n.icon]}
                                        </tspan>
                                        <tspan x={0} y={18} className='type'>
                                            {n.type}
                                        </tspan>
                                    </text>
                                </g>
                                <g className='label' transform={`translate(0,${nodeSize})`}>
                                    <text>
                                        {n.touch} {n.label}
                                    </text>
                                </g>
                            </g>
                        ))}
                </g>
            </svg>
        );
    }
}
