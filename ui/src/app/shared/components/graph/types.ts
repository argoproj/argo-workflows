export type Node = string;
export type NodeGroup = string;

export interface NodeLabel {
    type: string;
    label: string;
    icon?: string;
    classNames?: string;
    x?: number;
    y?: number;
}

export interface EdgeLabel {
    label?: string;
    classNames?: string;
    points?: {x: number; y: number}[];
}

export interface Graph {
    nodes: Map<Node, NodeLabel>;
    edges: Map<Edge, EdgeLabel>;
    nodeGroups: Map<NodeGroup, Set<Node>>;
    width?: number;
    height?: number;
}

export interface Edge {
    v: string;
    w: string;
}
