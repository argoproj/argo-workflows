export type node = string;

export interface Edge {
    v: node;
    w: node;
}

export class Graph {
    public nodes: node[] = [];
    public edges: Set<Edge> = new Set<Edge>();

    public removeTransitives() {
        this.nodes.forEach(a => {
            this.nodes
                .filter(b => this.edgeExists(a, b))
                .forEach(b => {
                    this.nodes.filter(c => this.edgeExists(b, c)).forEach(c => this.removeEdge(a, c));
                });
        });
    }

    public outgoingEdges(v: node) {
        const edges: node[] = [];
        this.edges.forEach(e => {
            if (v === e.v) {
                edges.push(e.w);
            }
        });
        return edges;
    }

    public incomingEdges(w: node) {
        const edges: node[] = [];
        this.edges.forEach(e => {
            if (e.w === w) {
                edges.push(e.v);
            }
        });
        return edges;
    }

    private edgeExists(v: node, w: node) {
        return this.edges.has({v, w});
    }

    private removeEdge(v: node, w: node) {
        this.edges.delete({v, w});
    }
}
