export function shortNodeName(node: {name: string; displayName: string}): string {
    return node.displayName || node.name;
}
