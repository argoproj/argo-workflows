import {Checkbox} from 'argo-ui/src/components/checkbox';
import {DropDown} from 'argo-ui/src/components/dropdown/dropdown';
import * as classNames from 'classnames';
import * as React from 'react';
import {GraphIcon} from './icon';
import {formatLabel} from './label';
import {layout} from './layout';
import {Graph, Node} from './types';

require('./graph-panel.scss');

interface Filter {
    types: Set<string>;
    classNames: Set<string>;
}

interface State {
    nodeSize: number;
    horizontal: boolean;
    fast: boolean;
    filter: Filter;
}

interface Props {
    graph: Graph;
    filter: Filter;
    options?: React.ReactNode;
    nodeSize?: number;
    horizontal?: boolean;
    onSelect?: (id: string) => void;
}

export class GraphPanel extends React.Component<Props, State> {
    constructor(props: Readonly<Props>) {
        super(props);
        this.state = {
            nodeSize: props.nodeSize || 64,
            horizontal: props.horizontal,
            fast: false,
            filter: {types: new Set(this.props.filter.types), classNames: new Set(this.props.filter.classNames)}
        };
    }

    public render() {
        const nodeSize = this.state.nodeSize;
        layout(this.props.graph, nodeSize, this.state.horizontal, id => !this.visible(id), this.state.fast);
        const width = this.props.graph.width;
        const height = this.props.graph.height;
        return (
            <div>
                <div className='graph-options-panel'>
                    <DropDown
                        isMenu={true}
                        anchor={() => (
                            <div
                                className={classNames('top-bar__filter', {
                                    'top-bar__filter--selected':
                                        this.props.filter.types.size > this.state.filter.types.size || this.props.filter.classNames.size > this.state.filter.classNames.size
                                })}>
                                <i className='argo-icon-filter' aria-hidden='true' />
                                <i className='fa fa-angle-down' aria-hidden='true' />
                            </div>
                        )}>
                        <p>Types</p>
                        <ul>
                            {Array.from(this.props.filter.types)
                                .sort()
                                .map(x => (
                                    <li key={x} className='top-bar__filter-item'>
                                        <label>
                                            <Checkbox
                                                checked={this.state.filter.types.has(x)}
                                                onChange={checked => {
                                                    this.setState(s => {
                                                        const filter = s.filter;
                                                        if (checked) {
                                                            filter.types.add(x);
                                                        } else {
                                                            filter.types.delete(x);
                                                        }
                                                        return {filter: {...filter}};
                                                    });
                                                }}
                                            />{' '}
                                            {x}
                                        </label>
                                    </li>
                                ))}
                        </ul>
                        <p>Classes</p>
                        <ul>
                            {Array.from(this.props.filter.classNames)
                                .sort()
                                .map(x => (
                                    <li key={x} className='top-bar__filter-item'>
                                        <label>
                                            <Checkbox
                                                checked={this.state.filter.classNames.has(x)}
                                                onChange={checked => {
                                                    this.setState(s => {
                                                        const filter = s.filter;
                                                        if (checked) {
                                                            filter.classNames.add(x);
                                                        } else {
                                                            filter.classNames.delete(x);
                                                        }
                                                        return {filter: {...filter}};
                                                    });
                                                }}
                                            />{' '}
                                            {x}
                                        </label>
                                    </li>
                                ))}
                        </ul>
                    </DropDown>
                    <a onClick={() => this.setState(s => ({horizontal: !s.horizontal}))} title='Horizontal/vertical layout'>
                        <i className={`fa ${this.state.horizontal ? 'fa-long-arrow-alt-right' : 'fa-long-arrow-alt-down'}`} />
                    </a>
                    <a onClick={() => this.setState(s => ({nodeSize: s.nodeSize * 1.2}))} title='Zoom in'>
                        <i className='fa fa-search-plus' />
                    </a>
                    <a onClick={() => this.setState(s => ({nodeSize: s.nodeSize / 1.2}))} title='Zoom out'>
                        <i className='fa fa-search-minus' />
                    </a>
                    <a onClick={() => this.setState(s => ({fast: !s.fast}))} title='Use faster, but less pretty rendered' className={this.state.fast ? 'active' : ''}>
                        <i className='fa fa-bolt' />
                    </a>
                    {this.props.options}
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
                            {Array.from(this.props.graph.edges)
                                .filter(([, label]) => label.points)
                                .map(([e, label]) => (
                                    <g key={`edge/${e.v}/${e.w}`} className={`edge ${label.classNames || 'arrow'}`}>
                                        <path d={label.points.map((p, j) => (j === 0 ? `M ${p.x} ${p.y} ` : `L ${p.x} ${p.y}`)).join(' ')} className='line' />
                                        <g transform={`translate(${label.points[label.points.length === 1 ? 0 : 1].x},${label.points[label.points.length === 1 ? 0 : 1].y})`}>
                                            <text className='label' style={{fontSize: nodeSize / 6}}>
                                                {formatLabel(label.label)}
                                            </text>
                                        </g>
                                    </g>
                                ))}
                            {Array.from(this.props.graph.nodes)
                                .filter(([n, label]) => label.x !== null && this.visible(n))
                                .map(([n, label]) => (
                                    <g key={`node/${n}`} transform={`translate(${label.x},${label.y})`} className='node'>
                                        <title>{n}</title>
                                        <g className={`icon ${label.classNames || ''}`} onClick={() => this.props.onSelect && this.props.onSelect(n)}>
                                            <circle r={nodeSize / 2} className='bg' />
                                            <GraphIcon icon={label.icon} progress={label.progress} nodeSize={nodeSize} />
                                            <text y={nodeSize / 3} className='type' style={{fontSize: nodeSize / 5}}>
                                                {label.type}
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

    private visible(id: Node) {
        const label = this.props.graph.nodes.get(id);
        return this.state.filter.types.has(label.type) && Array.from(this.state.filter.classNames).find(className => (label.classNames || '').includes(className)) !== null;
    }
}
