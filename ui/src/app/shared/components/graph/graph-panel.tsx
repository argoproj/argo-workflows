import * as React from 'react';
import {useEffect} from 'react';
import {TextInput} from '../../../shared/components/text-input';
import {ScopedLocalStorage} from '../../scoped-local-storage';
import {FilterDropDown} from '../filter-drop-down';
import {Icon} from '../icon';
import {GraphIcon} from './icon';
import {formatLabel} from './label';
import {layout} from './layout';
import {Graph, Node} from './types';

require('./graph-panel.scss');

type IconShape = 'rect' | 'circle';

interface NodeGenres {
    [type: string]: boolean;
}

interface NodeClassNames {
    [type: string]: boolean;
}

interface NodeTags {
    [key: string]: boolean;
}

interface Props {
    graph: Graph;
    storageScope: string; // the scope of storage, similar graphs should use the same vaulue
    options?: React.ReactNode; // add to the option panel
    classNames?: string;
    nodeGenresTitle: string;
    nodeGenres: NodeGenres;
    nodeClassNamesTitle?: string;
    nodeClassNames?: NodeClassNames;
    nodeTagsTitle?: string;
    nodeTags?: NodeTags;
    nodeSize?: number; // default "64"
    horizontal?: boolean; // default "false"
    hideNodeTypes?: boolean; // default "false"
    hideOptions?: boolean; // default "false"
    defaultIconShape?: IconShape; // default "rect"
    iconShapes?: {[type: string]: Icon};
    selectedNode?: Node;
    onNodeSelect?: (id: Node) => void;
}

const merge = (a: {[key: string]: boolean}, b: {[key: string]: boolean}) => b && Object.assign(Object.assign({}, b), a);

export const GraphPanel = (props: Props) => {
    const storage = new ScopedLocalStorage('graph/' + props.storageScope);
    const [nodeSize, setNodeSize] = React.useState<number>(storage.getItem('nodeSize', props.nodeSize));
    const [horizontal, setHorizontal] = React.useState<boolean>(storage.getItem('horizontal', props.horizontal));
    const [fast, setFast] = React.useState<boolean>(storage.getItem('fast', false));
    const [nodeGenres, setNodeGenres] = React.useState<NodeGenres>(storage.getItem('nodeGenres', props.nodeGenres));
    const [nodeClassNames, setNodeClassNames] = React.useState<NodeClassNames>(storage.getItem('nodeClassNames', props.nodeClassNames));
    const [nodeTags, setNodeTags] = React.useState<NodeTags>(props.nodeTags);
    const [nodeSearchKeyword, setNodeSearchKeyword] = React.useState<string>('');

    useEffect(() => storage.setItem('nodeSize', nodeSize, props.nodeSize), [nodeSize]);
    useEffect(() => storage.setItem('horizontal', horizontal, props.horizontal), [horizontal]);
    useEffect(() => storage.setItem('fast', fast, false), [fast]);
    useEffect(() => storage.setItem('nodeGenres', nodeGenres, props.nodeGenres), [nodeGenres, props.nodeGenres]);
    useEffect(() => storage.setItem('nodeClassNames', nodeClassNames, props.nodeClassNames), [nodeClassNames, props.nodeClassNames]);

    // we must make sure we have all values in the state, this can change between renders
    // so this code patches them up
    useEffect(() => setNodeGenres(merge(nodeGenres, props.nodeGenres)), [props.nodeGenres]);
    useEffect(() => setNodeClassNames(merge(nodeClassNames, props.nodeClassNames)), [props.nodeClassNames]);
    useEffect(() => setNodeTags(merge(nodeTags, props.nodeTags)), [props.nodeTags]);

    const visible = (id: Node) => {
        const label = props.graph.nodes.get(id);
        // If the node matches the search string, return without considering filters
        if (nodeSearchKeyword && label.label.includes(nodeSearchKeyword)) {
            return true;
        }
        // If the node doesn't match enabled genres, don't display
        if (!nodeGenres[label.genre]) {
            return false;
        }
        // If the node doesn't match enabled node class names, don't display
        if (nodeClassNames && !Object.entries(nodeClassNames).find(([className, checked]) => checked && (label.classNames || '').split(' ').includes(className))) {
            return false;
        }
        // If the node doesn't match enabled node tags, don't display
        if (nodeTags && !Object.entries(nodeTags).find(([tag, checked]) => !label.tags || (checked && label.tags.has(tag)))) {
            return false;
        }
        return true;
    };

    layout(props.graph, nodeSize, horizontal, id => !visible(id), fast);
    const width = props.graph.width;
    const height = props.graph.height;

    return (
        <div>
            {!props.hideOptions && (
                <div className='graph-options-panel'>
                    <FilterDropDown
                        sections={[
                            {
                                title: props.nodeGenresTitle,
                                values: nodeGenres,
                                onChange: (label, checked) => {
                                    setNodeGenres(v => {
                                        v[label] = checked;
                                        return Object.assign({}, v);
                                    });
                                }
                            },
                            {
                                title: props.nodeClassNamesTitle,
                                values: nodeClassNames,
                                onChange: (label, checked) => {
                                    setNodeClassNames(v => {
                                        v[label] = checked;
                                        return Object.assign({}, v);
                                    });
                                }
                            },
                            {
                                title: props.nodeTagsTitle,
                                values: nodeTags,
                                onChange: (label, checked) => {
                                    setNodeTags(v => {
                                        v[label] = checked;
                                        return Object.assign({}, v);
                                    });
                                }
                            }
                        ]}
                    />
                    <a onClick={() => setHorizontal(s => !s)} title='Horizontal/vertical layout'>
                        <i className={`fa ${horizontal ? 'fa-long-arrow-alt-right' : 'fa-long-arrow-alt-down'} fa-fw`} />
                    </a>
                    <a onClick={() => setNodeSize(s => s * 1.2)} title='Zoom in'>
                        <i className='fa fa-search-plus fa-fw' />
                    </a>
                    <a onClick={() => setNodeSize(s => s / 1.2)} title='Zoom out'>
                        <i className='fa fa-search-minus fa-fw' />
                    </a>
                    <a onClick={() => setFast(s => !s)} title='Use faster, but less pretty renderer' className={fast ? 'active' : ''}>
                        <i className='fa fa-bolt fa-fw' />
                    </a>
                    {props.options}
                    <div className='node-search-bar'>
                        <TextInput value={nodeSearchKeyword} onChange={v => setNodeSearchKeyword(v)} placeholder={'Search'} />
                    </div>
                </div>
            )}
            <div className={'graph ' + props.classNames} style={{paddingTop: 35}}>
                {props.graph.nodes.size === 0 ? (
                    <p>Nothing to show</p>
                ) : (
                    <svg key='graph' width={width + nodeSize * 2} height={height + nodeSize * 2}>
                        <defs>
                            <marker id='arrow' viewBox='0 0 10 10' refX={10} refY={5} markerWidth={nodeSize / 8} markerHeight={nodeSize / 8} orient='auto-start-reverse'>
                                <path d='M 0 0 L 10 5 L 0 10 z' className='arrow' />
                            </marker>
                        </defs>
                        <g transform={`translate(${nodeSize},${nodeSize})`}>
                            {Array.from(props.graph.nodeGroups).map(([g, nodes]) => {
                                const r: {x1: number; y1: number; x2: number; y2: number} = {
                                    x1: width,
                                    y1: height,
                                    x2: 0,
                                    y2: 0
                                };
                                nodes.forEach(n => {
                                    const l = props.graph.nodes.get(n);
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
                            {Array.from(props.graph.edges)
                                .filter(([, label]) => label.points)
                                .map(([e, label]) => (
                                    <g key={`edge/${e.v}/${e.w}`} className={`edge ${label.classNames !== undefined ? label.classNames : 'arrow'}`}>
                                        <path
                                            d={label.points.map((p, j) => (j === 0 ? `M ${p.x} ${p.y} ` : `L ${p.x} ${p.y}`)).join(' ')}
                                            className='line'
                                            strokeWidth={nodeSize / 32}
                                        />
                                        <g transform={`translate(${label.points[label.points.length === 1 ? 0 : 1].x},${label.points[label.points.length === 1 ? 0 : 1].y})`}>
                                            <text className='edge-label' fontSize={nodeSize / 6}>
                                                {formatLabel(label.label)}
                                            </text>
                                        </g>
                                    </g>
                                ))}
                            {Array.from(props.graph.nodes)
                                .filter(([n, label]) => label.x !== null && visible(n))
                                .map(([n, label]) => (
                                    <g key={`node/${n}`} transform={`translate(${label.x},${label.y})`}>
                                        <title>{n}</title>
                                        <g
                                            className={`node ${label.classNames || ''} ${props.selectedNode === n ? ' selected' : ''}`}
                                            style={{strokeWidth: nodeSize / 15}}
                                            onClick={() => props.onNodeSelect && props.onNodeSelect(n)}>
                                            {((props.iconShapes || {})[label.genre] || props.defaultIconShape) === 'circle' ? (
                                                <circle r={nodeSize / 2} className='bg' />
                                            ) : (
                                                <rect x={-nodeSize / 2} y={-nodeSize / 2} width={nodeSize} height={nodeSize} className='bg' rx={nodeSize / 4} />
                                            )}
                                            <GraphIcon icon={label.icon} progress={label.progress} nodeSize={nodeSize} />
                                            {props.hideNodeTypes || (
                                                <text y={nodeSize * 0.33} className='type' fontSize={(12 * nodeSize) / GraphPanel.defaultProps.nodeSize}>
                                                    {label.genre}
                                                </text>
                                            )}
                                        </g>
                                        <g transform={`translate(0,${(nodeSize * 3) / 4})`}>
                                            <text className='node-label' fontSize={(18 * nodeSize) / GraphPanel.defaultProps.nodeSize}>
                                                {formatLabel(label.label)}
                                            </text>
                                        </g>
                                    </g>
                                ))}
                        </g>
                    </svg>
                )}
            </div>
        </div>
    );
};

GraphPanel.defaultProps = {
    nodeSize: 64
};
