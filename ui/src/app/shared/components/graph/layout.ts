import {layoutGraphFast} from './fast-layout';
import {layoutGraphPretty} from './pretty-layout';
import {Graph, Node} from './types';

export const layout = (graph: Graph, nodeSize: number, horizontal: boolean, hidden: (id: Node) => boolean, fast: boolean) => {
    // TODO - we should not re-layout the graph if options have not changed
    // if (Array.from(graph.nodes).find(([, l]) => l.x === undefined)) {
    (fast ? layoutGraphFast : layoutGraphPretty)(graph, nodeSize, horizontal, hidden);
    // }
};
