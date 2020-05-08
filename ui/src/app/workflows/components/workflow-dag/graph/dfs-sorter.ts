import {Graph, node} from './graph';

export class DfsSorter {
    public graph: Graph;
    public sorted: node[] = [];
    public visiting: Set<node> = new Set<node>();
    public discovered: Set<node> = new Set<node>();

    constructor(g: Graph) {
        this.graph = g;
    }

    public sort() {
        this.graph.nodes.forEach(n => this.visit(n));
        return this.sorted.reverse();
    }

    private visit(n: node) {
        if (this.discovered.has(n)) {
            return;
        }
        if (this.visiting.has(n)) {
            throw new Error('cyclic graph');
        }
        this.visiting.add(n);
        this.graph.outgoingEdges(n).forEach(outgoing => this.visit(outgoing));
        this.discovered.add(n);
        this.visiting.delete(n);
        this.sorted.push(n);
    }
}
