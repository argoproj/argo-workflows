import {Graph, node} from './graph';

export class DfsSorter {
    private graph: Graph;
    private sorted: node[] = [];
    private discovered: Set<node> = new Set<node>();

    constructor(g: Graph) {
        this.graph = g;
    }

    public sort() {
        // Pre-order DFS sort
        this.graph.nodes.forEach(n => this.visit(n));
        return this.sorted.reverse();
    }

    private visit(n: node) {
        if (this.discovered.has(n)) {
            return;
        }
        this.graph.outgoingEdges(n).forEach(outgoing => this.visit(outgoing));
        this.discovered.add(n);
        this.sorted.push(n);
    }
}
