import {NodeType} from '../../../../../models';

interface CollapsedNode {
    kind: string;
    parent: string;
    message: string;
    type: NodeType;
}

export function getCollapsedNodeName(parent: string, message: string, type: NodeType): string {
    return JSON.stringify({kind: 'collapsed', parent, message, type} as CollapsedNode);
}

export function isCollapsedNode(id: string): boolean {
    try {
        return (JSON.parse(id) as CollapsedNode).kind === 'collapsed';
    } catch (e) {
        return false;
    }
}

export function getNodeParent(id: string): string {
    return (JSON.parse(id) as CollapsedNode).parent;
}

export function getMessage(id: string): string {
    return (JSON.parse(id) as CollapsedNode).message;
}
