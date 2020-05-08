export type node = string;

export interface edge {
    v: node;
    w: node;
}

export class graph {
    public nodes: node[];
    public edges: Set<edge>;

    constructor() {
        this.nodes = [];
        this.edges = new Set<edge>();
    }
    public removeTransitives() {
        this.nodes.forEach(a => {
            this.nodes
                .filter(b => this.edgeExists(a, b))
                .forEach(b => {
                    this.nodes.filter(c => this.edgeExists(b, c)).forEach(c => this.removeEdge(a, c));
                });
        });
    }

    public outgoingEdges(node: node) {
        const edges: node[] = [];
        this.edges.forEach(e => {
            if (node === e.v) {
                edges.push(e.w);
            }
        });
        return edges;
    }

    public incomingEdges(node: node) {
        const edges: node[] = [];
        this.edges.forEach(e => {
            if (e.w === node) {
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

export class coffmanGrahamSorter {
    public graph: graph;
    public width: number;

    constructor(graph: graph) {
        this.graph = graph;
        this.width = 99;
    }

    public sort() {
        this.graph.removeTransitives();
        const nodes = new dfsSorter(this.graph).sort();
        const layers = new Array<node[]>();
        const levels = new Map<node, number>();

        nodes.forEach(n => {
            let dependantLevel = -1;
            this.graph.incomingEdges(n).forEach(dependant => {
                const level = levels.get(dependant);
                if (level === null) {
                    throw new Error('dependency order');
                }
                if (level > dependantLevel) {
                    dependantLevel = level;
                }
            });
            let level = -1;
            if (dependantLevel < layers.length - 1) {
                for (let i = dependantLevel + 1; i < layers.length; i++) {
                    if (layers[i].length < this.width) {
                        level = i;
                        break;
                    }
                }
            }
            if (level === -1) {
                layers.push(new Array<node>(0));
                level = layers.length - 1;
            }
            layers[level].push(n);
            levels.set(n, level);
        });
        return layers;
    }
}

class dfsSorter {
    public graph: graph;
    public sorted: node[];
    public visiting: Set<node>;
    public discovered: Set<node>;

    constructor(graph: graph) {
        this.graph = graph;
        this.sorted = [];
        this.visiting = new Set<node>();
        this.discovered = new Set<node>();
    }

    public sort() {
        this.graph.nodes.forEach(node => this.visit(node));
        return this.sorted.reverse();
    }

    private visit(node: node) {
        if (this.discovered.has(node)) {
            return;
        }
        if (this.visiting.has(node)) {
            throw new Error('cyclic graph');
        }
        this.visiting.add(node);
        this.graph.outgoingEdges(node).forEach(outgoing => this.visit(outgoing));
        this.discovered.add(node);
        this.visiting.delete(node);
        this.sorted.push(node);
    }
}
