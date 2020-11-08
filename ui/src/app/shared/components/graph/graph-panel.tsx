import * as React from 'react';
import {icons} from './icons';
import {formatLabel} from './label';
import {dagreLayout} from './layout';
import {Graph} from './types';

require('./graph-panel.scss');

/*
To be as featureful as the DAG graph we'd need:

* Fast and Dagre layouts.
* Animated - and percentage completed - running nodes.
* Hidden nodes.

 */

interface State {
    nodeSize: number;
}

interface Props {
    graph: Graph;
    onSelect?: (id: string) => void;
}

export class GraphPanel extends React.Component<Props, State> {
    constructor(props: Readonly<Props>) {
        super(props);
        this.state = {nodeSize: 64};
    }

    public render() {
        const nodeSize = this.state.nodeSize;
        dagreLayout(this.props.graph, nodeSize);
        const width = this.props.graph.width;
        const height = this.props.graph.height;
        return (
            <div>
                <div className='graph-options-panel'>
                    <a onClick={() => this.setState(s => ({nodeSize: s.nodeSize * 1.2}))} title='Zoom in'>
                        <i className='fa fa-search-plus' />
                    </a>
                    <a onClick={() => this.setState(s => ({nodeSize: s.nodeSize / 1.2}))} title='Zoom out'>
                        <i className='fa fa-search-minus' />
                    </a>
                </div>
                <div className='graph'>
                    <svg key='graph' width={width + nodeSize * 2} height={height + nodeSize * 2}>
                        <defs>
                            <marker id='arrow' viewBox='0 0 10 10' refX={10} refY={5} markerWidth={nodeSize / 8} markerHeight={nodeSize / 8} orient='auto-start-reverse'>
                                <path d='M 0 0 L 10 5 L 0 10 z' className='arrow' />
                            </marker>
                        </defs>
                        <g transform={`translate(${nodeSize},${nodeSize})`}>
                            {Array.from(this.props.graph.nodeGroups).map(([g, nodes]) => {
                                const r: {x1: number; y1: number; x2: number; y2: number} = {
                                    x1: width,
                                    y1: height,
                                    x2: 0,
                                    y2: 0
                                };
                                nodes.forEach(n => {
                                    const l = this.props.graph.nodes.get(n);
                                    r.x1 = Math.min(r.x1, l.x);
                                    r.y1 = Math.min(r.y1, l.y);
                                    r.x2 = Math.max(r.x2, l.x);
                                    r.y2 = Math.max(r.y2, l.y);
                                });
                                return (
                                    <g key={`group/${g}`} className='group' transform={`translate(${r.x1 - nodeSize},${r.y1 - nodeSize})`}>
                                        <rect width={r.x2 - r.x1 + 2 * nodeSize} height={r.y2 - r.y1 + 2 * nodeSize} />
                                    </g>
                                );
                            })}
                            {Array.from(this.props.graph.edges).map(([e, label]) => (
                                <g key={`edge/${e.v}/${e.w}`} className={`edge ${label.classNames || 'arrow'}`}>
                                    <path d={label.points.map((p, j) => (j === 0 ? `M ${p.x} ${p.y} ` : `L ${p.x} ${p.y}`)).join(' ')} className='line' />
                                    <g transform={`translate(${label.points[1].x},${label.points[1].y})`}>
                                        <text className='label' style={{fontSize: nodeSize / 6}}>
                                            {formatLabel(label.label)}
                                        </text>
                                    </g>
                                </g>
                            ))}
                            {Array.from(this.props.graph.nodes)
                                .filter(([_, label]) => label.x)
                                .map(([n, label]) => (
                                    <g key={`node/${n}`} transform={`translate(${label.x},${label.y})`} className='node'>
                                        <title>{n}</title>
                                        <g className={`icon ${label.classNames || ''}`} onClick={() => this.props.onSelect && this.props.onSelect(n)}>
                                            <circle r={nodeSize / 2} className='bg' />
                                            <text>
                                                <tspan x={0} y={nodeSize / 16} className='icon' style={{fontSize: nodeSize / 2}}>
                                                    {icons[label.icon]}
                                                </tspan>
                                                <tspan x={0} y={nodeSize / 3.5} className='type' style={{fontSize: nodeSize / 5}}>
                                                    {label.type}
                                                </tspan>
                                            </text>
                                        </g>
                                        <g className='label' transform={`translate(0,${(nodeSize * 3) / 4})`}>
                                            <text style={{fontSize: nodeSize / 5}}>{formatLabel(label.label)}</text>
                                        </g>
                                    </g>
                                ))}
                        </g>
                    </svg>
                </div>
            </div>
        );
    }
}
