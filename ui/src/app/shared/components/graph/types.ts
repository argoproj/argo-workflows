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

    public removeTransitives() {
        this.nodes.forEach((aLabel, a) => {
            this.nodes.forEach((bLabel, b) => {
                if (this.edgeExists(a, b)) {
                    this.nodes.forEach((cLabel, c) => {
                        if (this.edgeExists(b, c)) {
                            this.removeEdge(a, c);
                        }
                    });
                }
            });
        });
    }

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

    private edgeExists(v: Node, w: Node) {
        return this.edges.has({v, w});
    }

    private removeEdge(v: Node, w: Node) {
        this.edges.delete({v, w});
    }
}
