import {layoutGraphFast} from './fast-layout';
import {layoutGraphPretty} from './pretty-layout';
import {Graph, Node} from './types';

export const layout = (graph: Graph, nodeSize: number, horizontal: boolean, hidden: (id: Node) => boolean, fast: boolean) => {
    (fast ? layoutGraphFast : layoutGraphPretty)(graph, nodeSize, horizontal, hidden);
};
