import {DfsSorter} from './dfs-sorter';
import {Graph, Node} from './types';

export class CoffmanGrahamSorter {
    private graph: Graph;
    private width: number = 9999;

    constructor(g: Graph) {
        this.graph = g;
    }

    public sort() {
        // normally you should remove transitive here, but this is expensive, and we don't expect to find any
        // this.graph.removeTransitives();
        const nodes = new DfsSorter(this.graph).sort();
        const layers = new Array<Node[]>();
        const levels = new Map<Node, number>();

        nodes.forEach(n => {
            let dependantLevel = -1;
            this.graph.incomingEdges(n).forEach(dependant => {
                const l = levels.get(dependant);
                if (l === null) {
                    throw new Error('dependency order');
                }
                if (l > dependantLevel) {
                    dependantLevel = l;
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
                layers.push(new Array<Node>(0));
                level = layers.length - 1;
            }
            layers[level].push(n);
            levels.set(n, level);
        });
        return layers;
    }
}
