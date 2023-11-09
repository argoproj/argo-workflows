import {Icon} from '../icon';

export type Node = string;
type NodeGroup = string;

export interface NodeLabel {
    label: string; // the label of the node - placed below the icon
    genre: string; // the class or type of the node, displayed below the icon
    icon?: Icon;
    classNames?: string;
    tags?: Set<string>; // completely generic tags for filtering nodes
    progress?: number; // progress between 0..1
    x?: number;
    y?: number;
}

interface EdgeLabel {
    label?: string;
    classNames?: string;
    points?: {x: number; y: number}[];
}

export interface Edge {
    v: string;
    w: string;
}

export class Graph {
    public nodes: Map<Node, NodeLabel> = new Map();
    public edges: Map<Edge, EdgeLabel> = new Map();
    public nodeGroups: Map<NodeGroup, Set<Node>> = new Map();
    public width?: number;
    public height?: number;

    public outgoingEdges(v: Node) {
        const edges: Node[] = [];
        this.edges.forEach((_, e) => {
            if (v === e.v) {
                edges.push(e.w);
            }
        });
        return edges;
    }

    public incomingEdges(w: Node) {
        const edges: Node[] = [];
        this.edges.forEach((_, e) => {
            if (e.w === w) {
                edges.push(e.v);
            }
        });
        return edges;
    }
}
