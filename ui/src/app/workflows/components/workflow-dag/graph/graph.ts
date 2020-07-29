export type node = string;

export interface Edge {
    from: node;
    to: node;
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
            if (v === e.from) {
                edges.push(e.to);
            }
        });
        return edges;
    }

    public incomingEdges(w: node) {
        const edges: node[] = [];
        this.edges.forEach(e => {
            if (e.to === w) {
                edges.push(e.from);
            }
        });
        return edges;
    }

    private edgeExists(v: node, w: node) {
        return this.edges.has({from: v, to: w});
    }

    private removeEdge(v: node, w: node) {
        this.edges.delete({from: v, to: w});
    }
}
