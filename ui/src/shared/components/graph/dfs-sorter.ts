import {Graph, Node} from './types';

export class DfsSorter {
    private graph: Graph;
    private sorted: Node[] = [];
    private discovered: Set<Node> = new Set<Node>(); // Fully visited
    private visiting: Set<Node> = new Set<Node>(); // Currently in DFS call stack

    constructor(g: Graph) {
        this.graph = g;
    }

    public sort() {
        // Pre-order DFS sort
        this.graph.nodes.forEach((_, n) => this.visit(n));
        return this.sorted.reverse();
    }

    private visit(n: Node) {
        if (this.discovered.has(n)) {
            return;
        }
        if (this.visiting.has(n)) {
            // Cycle detected, prevent infinite recursion
            return;
        }

        this.visiting.add(n);
        this.graph.outgoingEdges(n).forEach(outgoing => this.visit(outgoing));
        this.visiting.delete(n);
        this.discovered.add(n);
        this.sorted.push(n);
    }
}
